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
	// Parent: supermarket polymarket
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
		Long: `Interactively prompts for Polymarket API credentials and saves them to
~/.supermodel/config.yaml (mode 0600, never committed to git).

Two credential tiers are supported:

  Builder API (Ed25519) — use --key-id + --secret-key
    Headers: X-PM-Access-Key, X-PM-Timestamp, X-PM-Signature

  CLOB API (HMAC-SHA256) — use --api-key + --secret [+ --passphrase]
    Headers: POLY_API_KEY, POLY_SIGNATURE, POLY_TIMESTAMP

Environment variable overrides:
  POLYMARKET_KEY_ID, POLYMARKET_SECRET_KEY (Builder)
  POLYMARKET_API_KEY, POLYMARKET_SECRET, POLYMARKET_PASSPHRASE (CLOB)
  POLYMARKET_PRIVATE_KEY, POLYMARKET_EOA_ADDRESS,
  POLYMARKET_PROXY_WALLET, POLYMARKET_DEPOSIT_WALLET (wallet)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolymarketAuth(cmd)
		},
	}
	// Builder API flags
	authCmd.Flags().String("key-id", "", "Builder API key ID (UUID)")
	authCmd.Flags().String("secret-key", "", "Builder API secret key (base64 Ed25519 private key)")
	// CLOB API flags
	authCmd.Flags().String("api-key", "", "CLOB API key (UUID)")
	authCmd.Flags().String("secret", "", "CLOB API secret (base64 HMAC key)")
	authCmd.Flags().String("passphrase", "", "CLOB API passphrase")
	// Wallet flags
	authCmd.Flags().String("private-key", "", "Ethereum private key (64 hex chars, for L1 signing)")
	authCmd.Flags().String("eoa-address", "", "EOA wallet address (0x…, derived from private key)")
	authCmd.Flags().String("proxy-wallet", "", "Proxy/Safe wallet address (0x…)")
	authCmd.Flags().String("deposit-wallet", "", "Deposit wallet address (0x…, USDC deposits)")
	authCmd.Flags().Int64("chain-id", 0, "Polygon chain ID (137=mainnet, 80002=Amoy testnet)")

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

	fmt.Fprintln(os.Stderr, "Configure Polymarket API credentials.")
	fmt.Fprintln(os.Stderr, "Press Enter to skip any field (existing values are preserved).")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "── Builder API (Ed25519) ──────────────")
	if err := polySetFlag(cmd, "key-id", "Builder Key ID (UUID): ", &poly.KeyID, false); err != nil {
		return err
	}
	if err := polySetFlag(cmd, "secret-key", "Builder Secret Key (base64 Ed25519): ", &poly.SecretKey, true); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "── CLOB API (HMAC-SHA256) ─────────────")
	if err := polySetFlag(cmd, "api-key", "CLOB API Key (UUID): ", &poly.APIKey, false); err != nil {
		return err
	}
	if err := polySetFlag(cmd, "secret", "CLOB API Secret: ", &poly.Secret, true); err != nil {
		return err
	}
	if err := polySetFlag(cmd, "passphrase", "CLOB API Passphrase: ", &poly.Passphrase, true); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "── Wallet ───────────────────────────")
	if err := polySetFlag(cmd, "private-key", "Private Key (64 hex chars): ", &poly.PrivateKey, true); err != nil {
		return err
	}
	if err := polySetFlag(cmd, "eoa-address", "EOA Address (0x…): ", &poly.EOAAddress, false); err != nil {
		return err
	}
	if err := polySetFlag(cmd, "proxy-wallet", "Proxy/Safe Wallet (0x…): ", &poly.ProxyWallet, false); err != nil {
		return err
	}
	if err := polySetFlag(cmd, "deposit-wallet", "Deposit Wallet (0x…): ", &poly.DepositWallet, false); err != nil {
		return err
	}

	if chainID, _ := cmd.Flags().GetInt64("chain-id"); chainID != 0 {
		poly.ChainID = chainID
	} else if poly.ChainID == 0 {
		fmt.Fprint(os.Stderr, "Chain ID (137=mainnet, 80002=testnet, Enter to skip): ")
		if line, e := polyReadLine(); e != nil {
			return e
		} else if line != "" {
			var id int64
			if _, err := fmt.Sscanf(line, "%d", &id); err == nil {
				poly.ChainID = id
			}
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}
	ui.Success("Polymarket credentials saved to %s", config.Path())
	return nil
}

// polySetFlag writes the flag value (if set) or prompts interactively.
// Existing values in *dest are preserved when the user presses Enter.
func polySetFlag(cmd *cobra.Command, flag, label string, dest *string, secret bool) error {
	if v, _ := cmd.Flags().GetString(flag); v != "" {
		*dest = v
		return nil
	}
	var val string
	var err error
	if secret {
		val, err = polyPromptSecret(label)
	} else {
		fmt.Fprint(os.Stderr, label)
		val, err = polyReadLine()
	}
	if err != nil {
		return err
	}
	if val != "" {
		*dest = val
	}
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
