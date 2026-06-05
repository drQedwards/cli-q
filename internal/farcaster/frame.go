package farcaster

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/supermodeltools/cli/internal/polymarket"
)

// BuildFrame returns the HTML for a Farcaster Frame showing a Polymarket market.
// postURL is the absolute URL that Warpcast will POST button presses to.
func BuildFrame(market *polymarket.Market, postURL string) string {
	imageURL := "https://og.polymarket.com/market/" + market.ConditionID + ".png"

	yesPrice, noPrice := tokenPrices(market.Tokens)
	title := fmt.Sprintf("%s — YES: %s | NO: %s", truncateFrame(market.Question, 80), yesPrice, noPrice)

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>%s</title>
  <meta property="og:title" content="%s" />
  <meta property="og:image" content="%s" />
  <meta property="fc:frame" content="vNext" />
  <meta property="fc:frame:image" content="%s" />
  <meta property="fc:frame:button:1" content="Refresh" />
  <meta property="fc:frame:post_url" content="%s" />
</head>
<body>
  <h1>%s</h1>
  <p>YES: %s &nbsp; NO: %s</p>
</body>
</html>`,
		html.EscapeString(market.Question),
		html.EscapeString(title),
		html.EscapeString(imageURL),
		html.EscapeString(imageURL),
		html.EscapeString(postURL),
		html.EscapeString(market.Question),
		html.EscapeString(yesPrice),
		html.EscapeString(noPrice),
	)
}

func tokenPrices(tokens []polymarket.Token) (yes, no string) {
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

func truncateFrame(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
