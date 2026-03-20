package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jclement/deaddrop/internal/crypto"
	"github.com/jclement/deaddrop/internal/encode"
	"github.com/jclement/deaddrop/internal/pdf"
	"github.com/jclement/deaddrop/internal/ui"
)

func restoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [image...]",
		Short: "Decrypt a secret from a QR code image or Z85 text",
		Long: `Restore a secret from a dead drop.

Input sources:
  - Image file arguments (PNG/JPEG): scans QR codes and reassembles multi-page payloads
  - "-" argument: reads raw ciphertext from stdin
  - No argument: prompts to paste Z85-encoded text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args)
		},
	}
	return cmd
}

func runRestore(args []string) error {
	// 1. Obtain the payload
	payload, err := readPayload(args)
	if err != nil {
		return err
	}

	// 2. Prompt for passphrase
	passphrase, err := promptPassphrase()
	if err != nil {
		return err
	}
	defer zeroString(&passphrase)

	// 3. Decrypt
	secret, err := crypto.Decrypt(payload, passphrase)
	if err != nil {
		return fmt.Errorf("decryption failed (wrong passphrase?): %w", err)
	}

	// 4. Output
	if !quiet {
		fmt.Fprintln(os.Stderr, ui.StyleSuccess.Render("Decrypted successfully."))
		fmt.Fprintln(os.Stderr)
	}
	os.Stdout.Write(secret)

	return nil
}

func readPayload(args []string) ([]byte, error) {
	if len(args) >= 1 && args[0] != "-" {
		// Image file(s): scan QR code(s)
		chunks := make([][]byte, 0, len(args))
		for _, path := range args {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Scanning QR code from %s...\n", path)
			}
			data, err := pdf.DecodeQR(path)
			if err != nil {
				return nil, fmt.Errorf("scanning QR code from %s: %w", path, err)
			}
			chunks = append(chunks, data)
		}
		if !quiet {
			fmt.Fprintf(os.Stderr, ui.StyleSuccess.Render("Scanned %d QR code(s).")+"\n", len(chunks))
		}
		return pdf.ReassemblePayload(chunks)
	}

	if len(args) == 1 && args[0] == "-" {
		// Raw ciphertext from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		return data, nil
	}

	// No args: prompt for Z85 text or check if stdin is piped
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		// Try Z85 decode first, fall back to raw
		decoded, z85Err := encode.DecodeZ85(strings.TrimSpace(string(data)))
		if z85Err == nil {
			return decoded, nil
		}
		return data, nil
	}

	// Interactive: prompt for Z85 text
	var z85Input string
	err := huh.NewText().
		Title("Paste the Z85-encoded text from the PDF").
		Description("Copy the encoded payload block and paste it here").
		Value(&z85Input).
		Run()
	if err != nil {
		return nil, fmt.Errorf("reading Z85 text: %w", err)
	}

	// Strip whitespace/newlines from the pasted block
	cleaned := strings.Join(strings.Fields(z85Input), "")
	decoded, err := encode.DecodeZ85(cleaned)
	if err != nil {
		return nil, fmt.Errorf("decoding Z85 text: %w", err)
	}
	return decoded, nil
}

func promptPassphrase() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("passphrase required but stdin is not a terminal; pipe passphrase via a different mechanism")
	}

	var passphrase string
	err := huh.NewInput().
		Title("Enter passphrase").
		Description("The passphrase handwritten on the printed page").
		EchoMode(huh.EchoModePassword).
		Value(&passphrase).
		Run()
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}
	return passphrase, nil
}
