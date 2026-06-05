// Package polymarket provides a client and graph renderer for the
// Polymarket CLOB (Central Limit Order Book) API.
//
// It fetches prediction market data — markets, outcomes, prices, and
// volume — and renders it using Supermodel's human/json/dot output formats.
//
// Credentials are read from *config.Config; set them interactively with:
//
//	supermodel polymarket auth
//
// or via environment variables (POLYMARKET_API_KEY, POLYMARKET_SECRET,
// POLYMARKET_PASSPHRASE, POLYMARKET_PRIVATE_KEY, POLYMARKET_PROXY_WALLET).
// Public market data endpoints work without credentials.
package polymarket
