package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/supermodeltools/cli/internal/config"
	"github.com/supermodeltools/cli/internal/farcaster"
	"github.com/supermodeltools/cli/internal/polymarket"
	"github.com/supermodeltools/cli/internal/ui"
)

// castSummary formats a brief cast text for a single market (≤320 chars).
func castSummary(m *polymarket.Market) string {
	yesPrice, noPrice := castTokenPrices(m.Tokens)
	id := m.ConditionID
	if len(id) > 10 {
		id = id[:10]
	}
	q := m.Question
	// keep question short enough to fit with prices under 320 chars
	maxQ := 240
	if len([]rune(q)) > maxQ {
		q = string([]rune(q)[:maxQ-1]) + "…"
	}
	return fmt.Sprintf("%s\nYES: %s | NO: %s\nID: %s", q, yesPrice, noPrice, id)
}

// castTokenPrices returns formatted YES/NO prices for a market.
func castTokenPrices(tokens []polymarket.Token) (yes, no string) {
	for _, t := range tokens {
		switch strings.ToUpper(t.Outcome) {
		case "YES":
			yes = strconv.FormatFloat(t.Price, 'f', 4, 64)
		case "NO":
			no = strconv.FormatFloat(t.Price, 'f', 4, 64)
		}
	}
	if yes == "" {
		yes = "-"
	}
	if no == "" {
		no = "-"
	}
	return
}

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

	// supermodel polymarket cast
	var castMarket, castChannel string
	var castLimit int
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast Polymarket odds to Farcaster",
		Long: `Fetches Polymarket market data and casts a summary to Farcaster via Neynar.

If --market is given, casts a single market's odds.
Otherwise, fetches the top --limit markets by volume and casts a summary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolymarketCast(cmd.Context(), castMarket, castChannel, castLimit)
		},
	}
	castCmd.Flags().StringVar(&castMarket, "market", "", "condition_id of a single market to cast")
	castCmd.Flags().StringVar(&castChannel, "channel", "", "Farcaster channel slug (overrides config default)")
	castCmd.Flags().IntVar(&castLimit, "limit", 5, "number of top markets to cast when --market is not set")

	// supermodel polymarket watch
	var watchMarket, watchChannel string
	var watchThreshold float64
	var watchInterval int
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch a Polymarket market and cast on significant price moves",
		Long: `Polls a Polymarket market every --interval seconds. When the YES price
moves by at least --threshold percent, casts an alert to Farcaster.

Press Ctrl-C to stop.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if watchMarket == "" {
				return fmt.Errorf("--market is required")
			}
			return runPolymarketWatch(cmd.Context(), watchMarket, watchChannel, watchThreshold, watchInterval)
		},
	}
	watchCmd.Flags().StringVar(&watchMarket, "market", "", "condition_id of market to watch (required)")
	watchCmd.Flags().StringVar(&watchChannel, "channel", "", "Farcaster channel slug (overrides config default)")
	watchCmd.Flags().Float64Var(&watchThreshold, "threshold", 3.0, "price-move threshold in percent before casting an alert")
	watchCmd.Flags().IntVar(&watchInterval, "interval", 30, "poll interval in seconds")

	// supermodel polymarket serve
	var servePort int
	var serveMarket string
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve a Farcaster Frame for a Polymarket market",
		Long: `Starts an HTTP server that serves a Farcaster Frame showing live
Polymarket data. The frame supports a "Refresh" button that re-fetches
market data on each button press.

GET / — serves the frame HTML
POST / — handles frame button presses and returns an updated frame`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serveMarket == "" {
				return fmt.Errorf("--market is required")
			}
			return runPolymarketServe(cmd.Context(), serveMarket, servePort)
		},
	}
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "HTTP port to listen on")
	serveCmd.Flags().StringVar(&serveMarket, "market", "", "condition_id of market to serve (required)")

	polyCmd.AddCommand(graphCmd)
	polyCmd.AddCommand(authCmd)
	polyCmd.AddCommand(castCmd)
	polyCmd.AddCommand(watchCmd)
	polyCmd.AddCommand(serveCmd)
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

// runPolymarketCast fetches market data and casts to Farcaster.
func runPolymarketCast(ctx context.Context, market, channel string, limit int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.RequireFarcasterSigner(); err != nil {
		return err
	}
	fc := cfg.EnsureFarcaster()
	if channel == "" {
		channel = fc.Channel
	}

	client := polymarket.New(cfg)
	fcClient := farcaster.New(fc.NeynarAPIKey)

	var markets []polymarket.Market
	if market != "" {
		m, err := client.GetMarket(ctx, market)
		if err != nil {
			return fmt.Errorf("fetch market %s: %w", market, err)
		}
		markets = []polymarket.Market{*m}
	} else {
		s := ui.Start("Fetching Polymarket markets…")
		fetched, _, err := client.ListMarkets(ctx, limit, "")
		s.Stop()
		if err != nil {
			return fmt.Errorf("list markets: %w", err)
		}
		markets = fetched
		sort.Slice(markets, func(i, j int) bool {
			vi, _ := strconv.ParseFloat(markets[i].Volume, 64)
			vj, _ := strconv.ParseFloat(markets[j].Volume, 64)
			return vi > vj
		})
		if len(markets) > limit {
			markets = markets[:limit]
		}
	}

	for _, m := range markets {
		text := castSummary(&m)
		req := farcaster.CastRequest{
			SignerUUID: fc.SignerUUID,
			Text:       text,
		}
		if channel != "" {
			req.ChannelID = channel
		}
		resp, err := fcClient.Cast(req)
		if err != nil {
			return fmt.Errorf("cast market %s: %w", m.ConditionID, err)
		}
		ui.Success("Cast posted: %s", resp.Cast.Hash)
	}
	return nil
}

// runPolymarketWatch polls a market and casts an alert when the price moves.
func runPolymarketWatch(ctx context.Context, market, channel string, threshold float64, intervalSecs int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.RequireFarcasterSigner(); err != nil {
		return err
	}
	fc := cfg.EnsureFarcaster()
	if channel == "" {
		channel = fc.Channel
	}

	// Use signal.NotifyContext for clean Ctrl-C handling
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	polyClient := polymarket.New(cfg)
	fcClient := farcaster.New(fc.NeynarAPIKey)

	var lastYes float64
	initialized := false

	ticker := time.NewTicker(time.Duration(intervalSecs) * time.Second)
	defer ticker.Stop()

	fmt.Fprintf(os.Stderr, "Watching market %s (threshold: %.1f%%, interval: %ds)\n", market, threshold, intervalSecs)
	fmt.Fprintf(os.Stderr, "Press Ctrl-C to stop.\n")

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "\nStopped.")
			return nil
		case <-ticker.C:
			m, err := polyClient.GetMarket(ctx, market)
			if err != nil {
				fmt.Fprintf(os.Stderr, "poll error: %v\n", err)
				continue
			}
			yesStr, _ := castTokenPrices(m.Tokens)
			yesPrice, err := strconv.ParseFloat(yesStr, 64)
			if err != nil {
				continue
			}

			fmt.Fprintf(os.Stderr, "[%s] YES: %s\n", time.Now().Format("15:04:05"), yesStr)

			if !initialized {
				lastYes = yesPrice
				initialized = true
				continue
			}

			// Calculate percent change
			if lastYes != 0 {
				pct := ((yesPrice - lastYes) / lastYes) * 100
				if pct < 0 {
					pct = -pct
				}
				if pct >= threshold {
					direction := "↑"
					if yesPrice < lastYes {
						direction = "↓"
					}
					text := fmt.Sprintf("🚨 Polymarket Alert %s\n%s\nYES: %s → %s (%.1f%%)\nID: %s",
						direction,
						truncateCast(m.Question, 180),
						strconv.FormatFloat(lastYes, 'f', 4, 64),
						yesStr,
						((yesPrice-lastYes)/lastYes)*100,
						shortCastID(m.ConditionID),
					)
					req := farcaster.CastRequest{
						SignerUUID: fc.SignerUUID,
						Text:       text,
					}
					if channel != "" {
						req.ChannelID = channel
					}
					resp, err := fcClient.Cast(req)
					if err != nil {
						fmt.Fprintf(os.Stderr, "cast error: %v\n", err)
					} else {
						fmt.Fprintf(os.Stderr, "Alert cast: %s\n", resp.Cast.Hash)
					}
					lastYes = yesPrice
				}
			}
		}
	}
}

// runPolymarketServe starts an HTTP server that serves a Farcaster Frame.
func runPolymarketServe(ctx context.Context, market string, port int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	polyClient := polymarket.New(cfg)

	addr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(os.Stderr, "Serving Farcaster Frame at http://localhost%s\n", addr)
	fmt.Fprintf(os.Stderr, "Market: %s\n", market)
	fmt.Fprintf(os.Stderr, "Press Ctrl-C to stop.\n")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m, err := polyClient.GetMarket(r.Context(), market)
		if err != nil {
			http.Error(w, "failed to fetch market: "+err.Error(), http.StatusInternalServerError)
			return
		}
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		postURL := fmt.Sprintf("%s://%s/", scheme, r.Host)
		html := farcaster.BuildFrame(m, postURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Shutdown on context cancel
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func truncateCast(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func shortCastID(id string) string {
	if len(id) > 10 {
		return id[:10]
	}
	return id
}
