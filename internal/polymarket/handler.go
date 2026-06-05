package polymarket

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/supermodeltools/cli/internal/config"
	"github.com/supermodeltools/cli/internal/ui"
)

// Options configures the polymarket graph command.
type Options struct {
	Output   string // "human" | "json" | "dot"
	Limit    int    // max markets to fetch (0 defaults to 20)
	Category string // filter by category name or tag
	Market   string // fetch a single market by condition_id
}

// Run fetches Polymarket market data and renders it as a graph.
func Run(ctx context.Context, cfg *config.Config, opts Options) error {
	client := New(cfg)

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	var markets []Market
	if opts.Market != "" {
		m, err := client.GetMarket(ctx, opts.Market)
		if err != nil {
			return fmt.Errorf("fetch market %s: %w", opts.Market, err)
		}
		markets = []Market{*m}
	} else {
		s := ui.Start("Fetching Polymarket markets…")
		fetched, _, err := client.ListMarkets(ctx, limit, "")
		s.Stop()
		if err != nil {
			return fmt.Errorf("list markets: %w", err)
		}
		markets = fetched
	}

	if opts.Category != "" {
		cat := strings.ToLower(opts.Category)
		var filtered []Market
		for _, m := range markets {
			if strings.ToLower(m.Category) == cat || containsTag(m.Tags, cat) {
				filtered = append(filtered, m)
			}
		}
		markets = filtered
	}

	sort.Slice(markets, func(i, j int) bool {
		vi, _ := strconv.ParseFloat(markets[i].Volume, 64)
		vj, _ := strconv.ParseFloat(markets[j].Volume, 64)
		return vi > vj
	})

	return printGraph(os.Stdout, markets, opts)
}

func printGraph(w io.Writer, markets []Market, opts Options) error {
	switch opts.Output {
	case "json":
		return ui.JSON(w, markets)
	case "dot":
		return writeDOT(w, markets)
	default:
		return writeHuman(w, markets)
	}
}

func writeHuman(w io.Writer, markets []Market) error {
	rows := make([][]string, 0, len(markets))
	for _, m := range markets {
		status := "active"
		if m.Closed {
			status = "closed"
		}
		yesPrice, noPrice := tokenPrices(m.Tokens)
		rows = append(rows, []string{
			shortID(m.ConditionID),
			status,
			yesPrice,
			noPrice,
			truncate(m.Volume, 14),
			truncate(m.Question, 60),
		})
	}
	ui.Table(w, []string{"ID", "STATUS", "YES", "NO", "VOLUME", "QUESTION"}, rows)
	fmt.Fprintf(w, "\n%d markets\n", len(markets))
	return nil
}

func writeDOT(w io.Writer, markets []Market) error {
	fmt.Fprintln(w, "digraph polymarket {")
	fmt.Fprintln(w, "  rankdir=LR;")
	fmt.Fprintln(w, "  node [shape=box fontname=monospace];")

	for _, m := range markets {
		marketLabel := truncate(m.Question, 40)
		fmt.Fprintf(w, "  %q [label=%q tooltip=%q]\n", m.ConditionID, marketLabel, m.Category)
		for _, t := range m.Tokens {
			tokenLabel := fmt.Sprintf("%s @ %.4f", t.Outcome, t.Price)
			fmt.Fprintf(w, "  %q [label=%q shape=ellipse]\n", t.TokenID, tokenLabel)
			fmt.Fprintf(w, "  %q -> %q [label=%q]\n", m.ConditionID, t.TokenID, t.Outcome)
		}
	}

	fmt.Fprintln(w, "}")
	return nil
}

func tokenPrices(tokens []Token) (yes, no string) {
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

func shortID(id string) string {
	if len(id) > 10 {
		return id[:10]
	}
	return id
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.ToLower(t) == target {
			return true
		}
	}
	return false
}
