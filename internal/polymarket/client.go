package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/supermodeltools/cli/internal/config"
)

const defaultTimeout = 30 * time.Second

// Client is a Polymarket CLOB API client.
type Client struct {
	apiKey     string
	secret     string
	passphrase string
	baseURL    string
	http       *http.Client
}

// New returns a Client configured from cfg.
// Public endpoints work without credentials; authenticated endpoints
// require apiKey + secret + passphrase from cfg.Polymarket.
func New(cfg *config.Config) *Client {
	c := &Client{
		baseURL: config.PolymarketCLOBBase,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	if cfg.Polymarket != nil {
		c.apiKey = cfg.Polymarket.APIKey
		c.secret = cfg.Polymarket.Secret
		c.passphrase = cfg.Polymarket.Passphrase
	}
	return c
}

// ListMarkets returns up to limit active markets starting from cursor.
// Pass an empty cursor to start from the beginning.
func (c *Client) ListMarkets(ctx context.Context, limit int, cursor string) ([]Market, string, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("next_cursor", cursor)
	}

	path := "/markets"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp MarketList
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, "", err
	}
	return resp.Data, resp.NextCursor, nil
}

// GetMarket fetches a single market by condition_id.
func (c *Client) GetMarket(ctx context.Context, conditionID string) (*Market, error) {
	var m Market
	if err := c.get(ctx, "/markets/"+conditionID, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetPriceHistory returns price history for an outcome token.
// startTs and endTs are Unix timestamps; pass 0 to use server defaults.
func (c *Client) GetPriceHistory(ctx context.Context, tokenID string, startTs, endTs int64) ([]PricePoint, error) {
	params := url.Values{}
	params.Set("market", tokenID)
	params.Set("fidelity", "1")
	if startTs > 0 {
		params.Set("startTs", strconv.FormatInt(startTs, 10))
	}
	if endTs > 0 {
		params.Set("endTs", strconv.FormatInt(endTs, 10))
	}

	var resp PriceHistoryResponse
	if err := c.get(ctx, "/prices-history?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.History, nil
}

// GetOrderbook fetches the current order book for an outcome token.
func (c *Client) GetOrderbook(ctx context.Context, tokenID string) (*Orderbook, error) {
	var ob Orderbook
	if err := c.get(ctx, "/orderbook?token_id="+tokenID, &ob); err != nil {
		return nil, err
	}
	return &ob, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("POLY_API_KEY", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("polymarket GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read polymarket response: %w", err)
	}

	if resp.StatusCode >= 400 {
		snippet := string(body)
		if len([]rune(snippet)) > 200 {
			snippet = string([]rune(snippet)[:200]) + "…"
		}
		return fmt.Errorf("polymarket HTTP %d: %s", resp.StatusCode, snippet)
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode polymarket response from %s: %w", path, err)
		}
	}
	return nil
}
