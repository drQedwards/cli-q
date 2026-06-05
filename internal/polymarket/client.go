package polymarket

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/supermodeltools/cli/internal/config"
)

const defaultTimeout = 30 * time.Second

// Client is a Polymarket CLOB API client.
//
// Auth tiers (whichever credentials are populated):
//
//  1. Builder API (Ed25519)  — X-PM-Access-Key / X-PM-Timestamp / X-PM-Signature
//  2. CLOB API   (HMAC-SHA256) — POLY_API_KEY / POLY_SIGNATURE / POLY_TIMESTAMP / POLY_PASSPHRASE
type Client struct {
	// CLOB API (HMAC-SHA256)
	apiKey     string
	secret     string
	passphrase string

	// Builder API (Ed25519)
	keyID     string
	ed25519PK ed25519.PrivateKey // nil when not configured

	baseURL string
	http    *http.Client
}

// New returns a Client configured from cfg.
// Public endpoints work without credentials; authenticated endpoints require
// at least one credential set populated in cfg.Polymarket.
func New(cfg *config.Config) *Client {
	c := &Client{
		baseURL: config.PolymarketCLOBBase,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	if cfg.Polymarket == nil {
		return c
	}
	p := cfg.Polymarket
	c.apiKey = p.APIKey
	c.secret = p.Secret
	c.passphrase = p.Passphrase
	c.keyID = p.KeyID

	if p.SecretKey != "" {
		raw, err := base64.StdEncoding.DecodeString(p.SecretKey)
		if err != nil {
			// try URL-safe base64
			raw, err = base64.URLEncoding.DecodeString(p.SecretKey)
		}
		if err == nil && len(raw) == ed25519.PrivateKeySize {
			c.ed25519PK = ed25519.PrivateKey(raw)
		}
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
	c.addAuthHeaders(req, http.MethodGet, path, "")

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

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeaders(req, http.MethodPost, path, string(bodyBytes))

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("polymarket POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read polymarket response: %w", err)
	}

	if resp.StatusCode >= 400 {
		snippet := string(respBody)
		if len([]rune(snippet)) > 200 {
			snippet = string([]rune(snippet)[:200]) + "…"
		}
		return fmt.Errorf("polymarket HTTP %d: %s", resp.StatusCode, snippet)
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode polymarket response from %s: %w", path, err)
		}
	}
	return nil
}

// addAuthHeaders attaches whichever auth headers are available.
// Builder (Ed25519) takes precedence over CLOB (HMAC) when both are present.
func (c *Client) addAuthHeaders(req *http.Request, method, path, body string) {
	if c.ed25519PK != nil && c.keyID != "" {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		// Message: timestamp + METHOD + /path + body
		msg := ts + strings.ToUpper(method) + path + body
		sig := ed25519.Sign(c.ed25519PK, []byte(msg))
		req.Header.Set("X-PM-Access-Key", c.keyID)
		req.Header.Set("X-PM-Timestamp", ts)
		req.Header.Set("X-PM-Signature", base64.StdEncoding.EncodeToString(sig))
		return
	}
	if c.apiKey != "" {
		// CLOB L2 auth: HMAC-SHA256(secret, ts + method + path + body)
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sig := clobHMAC(c.secret, ts+strings.ToUpper(method)+path+body)
		req.Header.Set("POLY_API_KEY", c.apiKey)
		req.Header.Set("POLY_SIGNATURE", sig)
		req.Header.Set("POLY_TIMESTAMP", ts)
		if c.passphrase != "" {
			req.Header.Set("POLY_PASSPHRASE", c.passphrase)
		}
	}
}

// clobHMAC returns a base64-encoded HMAC-SHA256 of msg using key.
func clobHMAC(key, msg string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
