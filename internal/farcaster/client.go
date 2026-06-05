package farcaster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.neynar.com"
const defaultTimeout = 30 * time.Second

// Client wraps the Neynar REST API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New returns a Neynar API client authenticated with apiKey.
func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// SignerResponse is returned by CreateSigner and GetSigner.
type SignerResponse struct {
	SignerUUID        string `json:"signer_uuid"`
	PublicKey         string `json:"public_key"`
	Status            string `json:"status"` // "generated" | "pending_approval" | "approved"
	SignerApprovalURL string `json:"signer_approval_url"`
	FID               int64  `json:"fid"`
}

// CastRequest is the payload for POST /v2/farcaster/cast.
type CastRequest struct {
	SignerUUID string  `json:"signer_uuid"`
	Text       string  `json:"text"`
	ChannelID  string  `json:"channel_id,omitempty"`
	Embeds     []Embed `json:"embeds,omitempty"`
}

// Embed holds a URL embed for a cast.
type Embed struct {
	URL string `json:"url"`
}

// CastResponse is returned by Cast.
type CastResponse struct {
	Cast struct {
		Hash string `json:"hash"`
		Text string `json:"text"`
	} `json:"cast"`
}

// CreateSigner calls POST /v2/farcaster/signer and returns a new managed signer.
func (c *Client) CreateSigner() (*SignerResponse, error) {
	var out SignerResponse
	if err := c.post("/v2/farcaster/signer", nil, &out); err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}
	return &out, nil
}

// GetSigner calls GET /v2/farcaster/signer?signer_uuid=... and returns signer details.
func (c *Client) GetSigner(signerUUID string) (*SignerResponse, error) {
	var out SignerResponse
	if err := c.get("/v2/farcaster/signer?signer_uuid="+signerUUID, &out); err != nil {
		return nil, fmt.Errorf("get signer: %w", err)
	}
	return &out, nil
}

// Cast calls POST /v2/farcaster/cast with the given request body.
func (c *Client) Cast(req CastRequest) (*CastResponse, error) {
	var out CastResponse
	if err := c.post("/v2/farcaster/cast", req, &out); err != nil {
		return nil, fmt.Errorf("cast: %w", err)
	}
	return &out, nil
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("api_key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("neynar GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read neynar response: %w", err)
	}

	if resp.StatusCode >= 400 {
		snippet := string(body)
		if len([]rune(snippet)) > 200 {
			snippet = string([]rune(snippet)[:200]) + "…"
		}
		return fmt.Errorf("neynar HTTP %d: %s", resp.StatusCode, snippet)
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode neynar response from %s: %w", path, err)
		}
	}
	return nil
}

func (c *Client) post(path string, body any, out any) error {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("neynar POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read neynar response: %w", err)
	}

	if resp.StatusCode >= 400 {
		snippet := string(respBody)
		if len([]rune(snippet)) > 200 {
			snippet = string([]rune(snippet)[:200]) + "…"
		}
		return fmt.Errorf("neynar HTTP %d: %s", resp.StatusCode, snippet)
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode neynar response from %s: %w", path, err)
		}
	}
	return nil
}
