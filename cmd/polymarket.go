package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/supermodeltools/cli/internal/config"
	"github.com/supermodeltools/cli/internal/polymarket"
	"github.com/supermodeltools/cli/internal/ui"
)

func init() {
	// Parent: supermodel polymarket
	polyCmd := &cobra.Command{
		Use:   "polymarket",
		Short: "Polymarket prediction-market tools",
		Long:  `Fetch and visualise Polymarket prediction markets as dependency graphs.`,
	}

	// supermodel polymarket graph
	var graphOpts polymarket.Options
	graphCmd := &cobra.Command{
		Use:   "graph",
		Short: "Render Polymarket markets as a graph",
		Long: `Fetches live prediction-market data from the Polymarket CLOB API and
renders it as a graph.

Output formats:
  human — aligned table of markets (default)
  json  — full market JSON
  dot   — Graphviz DOT for use with dot/graphviz`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return polymarket.Run(cmd.Context(), cfg, graphOpts)
		},
	}
	graphCmd.Flags().StringVarP(&graphOpts.Output, "output", "o", "human", "output format: human|json|dot")
	graphCmd.Flags().IntVarP(&graphOpts.Limit, "limit", "n", 20, "max number of markets to fetch")
	graphCmd.Flags().StringVar(&graphOpts.Category, "category", "", "filter by category (e.g. sports, crypto, politics)")
	graphCmd.Flags().StringVar(&graphOpts.Market, "market", "", "fetch a single market by condition_id")

	// supermodel polymarket auth
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Configure Polymarket API credentials",
		Long: `Interactively prompts for Polymarket CLOB API credentials and saves
them to ~/.supermodel/config.yaml.

Get your credentials at https://polymarket.com → Settings → API Keys.

You can also skip prompts using flags or env vars:
  POLYMARKET_API_KEY, POLYMARKET_SECRET, POLYMARKET_PASSPHRASE,
  POLYMARKET_PRIVATE_KEY, POLYMARKET_PROXY_WALLET`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolymarketAuth(cmd)
		},
	}
	authCmd.Flags().String("api-key", "", "Polymarket API key (UUID)")
	authCmd.Flags().String("secret", "", "Polymarket API secret")
	authCmd.Flags().String("passphrase", "", "Polymarket API passphrase")
	authCmd.Flags().String("private-key", "", "Ethereum private key for L1 signing (optional)")
	authCmd.Flags().String("proxy-wallet", "", "Proxy wallet address (optional)")

	polyCmd.AddCommand(graphCmd)
	polyCmd.AddCommand(authCmd)
	rootCmd.AddCommand(polyCmd)
}

// runPolymarketAuth interactively collects and persists Polymarket credentials.
func runPolymarketAuth(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	poly := cfg.EnsurePolymarket()

	fmt.Fprintln(os.Stderr, "Configure Polymarket CLOB API credentials.")
	fmt.Fprintln(os.Stderr, "Get your keys at https://polymarket.com → Settings → API Keys.")
	fmt.Fprintln(os.Stderr)

	if v, _ := cmd.Flags().GetString("api-key"); v != "" {
		poly.APIKey = v
	} else if val, e := polyPromptSecret("API Key (UUID): "); e != nil {
		return e
	} else {
		poly.APIKey = val
	}

	if v, _ := cmd.Flags().GetString("secret"); v != "" {
		poly.Secret = v
	} else if val, e := polyPromptSecret("API Secret: "); e != nil {
		return e
	} else {
		poly.Secret = val
	}

	if v, _ := cmd.Flags().GetString("passphrase"); v != "" {
		poly.Passphrase = v
	} else if val, e := polyPromptSecret("API Passphrase: "); e != nil {
		return e
	} else {
		poly.Passphrase = val
	}

	if v, _ := cmd.Flags().GetString("private-key"); v != "" {
		poly.PrivateKey = v
	} else {
		fmt.Fprint(os.Stderr, "Private Key (optional, press Enter to skip): ")
		if val, e := polyReadLine(); e != nil {
			return e
		} else {
			poly.PrivateKey = val
		}
	}

	if v, _ := cmd.Flags().GetString("proxy-wallet"); v != "" {
		poly.ProxyWallet = v
	} else {
		fmt.Fprint(os.Stderr, "Proxy Wallet Address (optional, press Enter to skip): ")
		if val, e := polyReadLine(); e != nil {
			return e
		} else {
			poly.ProxyWallet = val
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}
	ui.Success("Polymarket credentials saved to %s", config.Path())
	return nil
}

// polyPromptSecret reads a sensitive value with echo suppression on TTYs.
func polyPromptSecret(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	fd := int(syscall.Stdin) //nolint:unconvert // syscall.Stdin is uintptr on Windows
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

// polyReadLine reads a single trimmed line from stdin.
func polyReadLine() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}
