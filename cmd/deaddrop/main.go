package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global flags
var (
	debug      bool
	quiet      bool
	jsonOutput bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "deaddrop",
		Short: "Print your secrets without trusting your printer",
		Long: `Dead Drop encrypts secrets and generates printable PDFs with QR codes.
The decryption key is handwritten after printing — the printer never sees the plaintext.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(restoreCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(updateCmd())

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("deaddrop %s (%s, %s)\n", version, commit, date))

	if err := rootCmd.Execute(); err != nil {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
