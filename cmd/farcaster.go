package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/supermodeltools/cli/internal/config"
	"github.com/supermodeltools/cli/internal/farcaster"
	"github.com/supermodeltools/cli/internal/ui"
)

func init() {
	// Parent: supermodel farcaster
	farcasterCmd := &cobra.Command{
		Use:   "farcaster",
		Short: "Farcaster integration tools",
		Long:  `Manage Farcaster credentials and post casts via the Neynar API.`,
	}

	// supermodel farcaster auth
	farcasterAuthCmd := &cobra.Command{
		Use:   "auth",
		Short: "Configure Farcaster/Neynar credentials",
		Long: `Interactively prompts for a Neynar API key, creates a managed signer,
and saves the approved credentials to ~/.supermodel/config.yaml.

Environment variable overrides:
  NEYNAR_API_KEY, FARCASTER_SIGNER_UUID, FARCASTER_FID, FARCASTER_CHANNEL`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFarcasterAuth()
		},
	}

	farcasterCmd.AddCommand(farcasterAuthCmd)
	rootCmd.AddCommand(farcasterCmd)
}

// runFarcasterAuth interactively collects Neynar credentials, creates a managed
// signer, polls for approval, and persists the result to config.
func runFarcasterAuth() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Configure Farcaster credentials via Neynar.")
	fmt.Fprintln(os.Stderr, "You will need a Neynar API key from https://neynar.com")
	fmt.Fprintln(os.Stderr)

	// Prompt for Neynar API key (echo-suppressed)
	apiKey, err := farcasterPromptSecret("Neynar API Key: ")
	if err != nil {
		return err
	}
	if apiKey == "" {
		return fmt.Errorf("Neynar API key is required")
	}

	client := farcaster.New(apiKey)

	// Create a new managed signer
	fmt.Fprintln(os.Stderr, "Creating managed signer…")
	signer, err := client.CreateSigner()
	if err != nil {
		return fmt.Errorf("create signer: %w", err)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Open this URL in Warpcast to approve:\n\n  %s\n\n", signer.SignerApprovalURL)
	fmt.Fprintln(os.Stderr, "Waiting for approval (up to 5 minutes)…")

	// Poll for signer approval (every 3 seconds, up to 5 minutes)
	spinner := ui.Start("Polling for signer approval…")

	timeout := time.After(5 * time.Minute)
	approved := false
	var finalSigner *farcaster.SignerResponse

	for !approved {
		select {
		case <-timeout:
			spinner.Stop()
			return fmt.Errorf("timed out waiting for signer approval — run `supermodel farcaster auth` again")
		default:
		}

		time.Sleep(3 * time.Second)

		updated, err := client.GetSigner(signer.SignerUUID)
		if err != nil {
			spinner.Stop()
			return fmt.Errorf("poll signer status: %w", err)
		}

		if updated.Status == "approved" {
			approved = true
			finalSigner = updated
		}
	}

	spinner.Stop()

	// Save credentials to config
	fc := cfg.EnsureFarcaster()
	fc.NeynarAPIKey = apiKey
	fc.SignerUUID = finalSigner.SignerUUID
	fc.FID = finalSigner.FID

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	ui.Success("Farcaster credentials saved to %s", config.Path())
	fmt.Fprintf(os.Stderr, "  Signer UUID: %s\n", finalSigner.SignerUUID)
	if finalSigner.FID != 0 {
		fmt.Fprintf(os.Stderr, "  FID: %d\n", finalSigner.FID)
	}
	return nil
}

// farcasterPromptSecret reads a sensitive value with echo suppression on TTYs.
func farcasterPromptSecret(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	fd := int(syscall.Stdin) //nolint:unconvert
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return polyReadLine()
}
