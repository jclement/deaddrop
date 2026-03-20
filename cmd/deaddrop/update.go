package main

import (
	"context"
	"fmt"
	"os"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"

	"github.com/jclement/deaddrop/internal/ui"
)

const repoSlug = "jclement/deaddrop"

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update deaddrop to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context())
		},
	}
}

func runUpdate(ctx context.Context) error {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return fmt.Errorf("creating update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{Source: source})
	if err != nil {
		return fmt.Errorf("creating updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}
	if !found {
		fmt.Println("No releases found.")
		return nil
	}

	if version != "dev" && !latest.GreaterThan(version) {
		fmt.Printf("Already up to date: %s\n", version)
		return nil
	}

	fmt.Printf("Updating %s -> %s\n", version, latest.Version())

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println(ui.StyleSuccess.Render("Updated successfully! Restart to use the new version."))
	return nil
}
