package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jclement/deaddrop/internal/crypto"
	"github.com/jclement/deaddrop/internal/encode"
	"github.com/jclement/deaddrop/internal/pdf"
	"github.com/jclement/deaddrop/internal/ui"
)

func createCmd() *cobra.Command {
	var (
		outputPath     string
		label          string
		title          string
		wordCount      int
		workFactor     int
		noInstructions bool
	)

	cmd := &cobra.Command{
		Use:   "create [file]",
		Short: "Encrypt a secret and generate a printable PDF",
		Long: `Encrypt a secret and generate a PDF with a QR code of the ciphertext.
The decryption passphrase is displayed on screen for you to handwrite on the printed page.

Input sources (in order of precedence):
  - File path argument: reads the file contents
  - "-" argument: reads from stdin
  - No argument + pipe: reads from stdin
  - No argument + TTY: interactive prompt`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args, createOptions{
				OutputPath:     outputPath,
				Label:          label,
				Title:          title,
				WordCount:      wordCount,
				WorkFactor:     workFactor,
				NoInstructions: noInstructions,
				JSON:           jsonOutput,
				Quiet:          quiet,
			})
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output PDF path (default: deaddrop-<timestamp>.pdf)")
	cmd.Flags().StringVarP(&label, "label", "l", "", "Label for the document (default: filename or \"secret\")")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Centered title on the PDF page")
	cmd.Flags().IntVarP(&wordCount, "words", "w", crypto.DefaultWordCount, "Number of diceware words (min: 5)")
	cmd.Flags().IntVar(&workFactor, "work-factor", crypto.DefaultWorkFactor, "age scrypt work factor")
	cmd.Flags().BoolVar(&noInstructions, "no-instructions", false, "Omit restore instructions from PDF")

	return cmd
}

type createOptions struct {
	OutputPath     string
	Label          string
	Title          string
	WordCount      int
	WorkFactor     int
	NoInstructions bool
	JSON           bool
	Quiet          bool
}

func runCreate(args []string, opts createOptions) error {
	// 1. Read the secret
	secret, sourceName, err := readSecret(args)
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}
	if len(secret) == 0 {
		return fmt.Errorf("secret is empty")
	}

	// 2. Determine label
	labelStr := opts.Label
	if labelStr == "" {
		labelStr = sourceName
	}

	// 3. Generate passphrase
	passphrase, err := crypto.GeneratePassphrase(opts.WordCount)
	if err != nil {
		return fmt.Errorf("generating passphrase: %w", err)
	}

	// 4. Encrypt
	payload, err := crypto.Encrypt(secret, passphrase, opts.WorkFactor)
	if err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	// 5. Generate Z85 text
	z85Text, err := encode.EncodeZ85(payload)
	if err != nil {
		return fmt.Errorf("Z85 encoding: %w", err)
	}

	// 6. Split payload if needed, generate QR for each chunk
	chunks := pdf.SplitPayload(payload)
	pages := make([]pdf.PageData, len(chunks))
	for i, chunk := range chunks {
		qrPNG, err := pdf.EncodeQR(chunk)
		if err != nil {
			return fmt.Errorf("generating QR code (page %d): %w", i+1, err)
		}

		// For multi-page, each page gets its own Z85 of its chunk
		pageZ85 := z85Text
		if len(chunks) > 1 {
			pageZ85, err = encode.EncodeZ85(chunk)
			if err != nil {
				return fmt.Errorf("Z85 encoding chunk %d: %w", i+1, err)
			}
		}

		pages[i] = pdf.PageData{
			QRCodePNG:  qrPNG,
			Z85Text:    pageZ85,
			PageNumber: i + 1,
			TotalPages: len(chunks),
		}
	}

	// 7. Determine output path
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = fmt.Sprintf("deaddrop-%s.pdf", time.Now().Format("20060102-150405"))
	}

	// 8. Generate PDF
	pdfOpts := pdf.PDFOptions{
		Label:            labelStr,
		Title:            opts.Title,
		Date:             time.Now(),
		OutputPath:       outputPath,
		ShowInstructions: !opts.NoInstructions,
	}
	if err := pdf.GeneratePDFToFile(pages, pdfOpts); err != nil {
		return fmt.Errorf("generating PDF: %w", err)
	}

	// 9. Output result
	if opts.JSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"pdf":           outputPath,
			"label":         labelStr,
			"title":         opts.Title,
			"passphrase":    passphrase,
			"words":         opts.WordCount,
			"algorithm":     "age-scrypt",
			"payload_bytes": len(payload),
			"pages":         len(pages),
		})
	}

	if !opts.Quiet {
		fmt.Println(ui.StyleSuccess.Render("Dead drop created."))
		fmt.Printf("  PDF:    %s\n", outputPath)
		fmt.Printf("  Label:  %s\n", labelStr)
		if opts.Title != "" {
			fmt.Printf("  Title:  %s\n", opts.Title)
		}
		fmt.Printf("  Pages:  %d\n", len(pages))
		fmt.Println()
		fmt.Println(ui.StyleWarning.Render("Write down this passphrase and destroy this terminal output:"))
		fmt.Println()
		fmt.Println(ui.StylePassphrase.Render(passphrase))
		fmt.Println()
	} else {
		// In quiet mode, still output the passphrase (it's essential)
		fmt.Println(passphrase)
	}

	return nil
}

func readSecret(args []string) ([]byte, string, error) {
	if len(args) == 1 {
		if args[0] == "-" {
			data, err := io.ReadAll(os.Stdin)
			return data, "stdin", err
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			return nil, "", err
		}
		return data, filepath.Base(args[0]), nil
	}

	// No args: check if stdin is a pipe
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		data, err := io.ReadAll(os.Stdin)
		return data, "stdin", err
	}

	// Interactive: use huh text input
	var secret string
	err := huh.NewText().
		Title("Enter your secret").
		Description("Paste or type the secret to encrypt (Ctrl+D or Esc to finish)").
		Value(&secret).
		Run()
	if err != nil {
		return nil, "", fmt.Errorf("reading secret: %w", err)
	}
	return []byte(secret), "secret", nil
}
