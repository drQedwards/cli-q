package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultAPIBase is the production Supermodel API endpoint.
const DefaultAPIBase = "https://api.supermodeltools.com"

// PolymarketCLOBBase is the Polymarket CLOB API endpoint.
const PolymarketCLOBBase = "https://clob.polymarket.com"

const defaultOutput = "human"

// PolymarketConfig holds Polymarket API credentials persisted under the
// polymarket: key in ~/.supermodel/config.yaml.
//
// Two auth tiers are supported:
//
//  1. CLOB API (HMAC-SHA256) — standard trading credentials
//     env: POLYMARKET_API_KEY, POLYMARKET_SECRET, POLYMARKET_PASSPHRASE
//
//  2. Builder API (Ed25519) — programmatic market-creation credentials
//     env: POLYMARKET_KEY_ID, POLYMARKET_SECRET_KEY
//
// Wallet env overrides: POLYMARKET_PRIVATE_KEY, POLYMARKET_EOA_ADDRESS,
// POLYMARKET_PROXY_WALLET, POLYMARKET_DEPOSIT_WALLET
type PolymarketConfig struct {
	// CLOB API credentials (HMAC-SHA256)
	APIKey     string `yaml:"api_key,omitempty"`
	Secret     string `yaml:"secret,omitempty"`
	Passphrase string `yaml:"passphrase,omitempty"`

	// Builder API credentials (Ed25519 — X-PM-* headers)
	KeyID     string `yaml:"key_id,omitempty"`
	SecretKey string `yaml:"secret_key,omitempty"` // base64-encoded Ed25519 private key (64 bytes)

	// Ethereum wallet
	PrivateKey    string `yaml:"private_key,omitempty"`    // 64-hex Ethereum private key
	EOAAddress    string `yaml:"eoa_address,omitempty"`    // checksummed EOA derived from private key
	ProxyWallet   string `yaml:"proxy_wallet,omitempty"`   // Gnosis Safe / proxy wallet
	DepositWallet string `yaml:"deposit_wallet,omitempty"` // UUPS deposit wallet for USDC

	// Network
	ChainID int64 `yaml:"chain_id,omitempty"` // 137 = Polygon mainnet, 80002 = Amoy testnet
}

// Config holds user-level settings persisted at ~/.supermodel/config.yaml.
type Config struct {
	APIKey     string            `yaml:"api_key,omitempty"`
	APIBase    string            `yaml:"api_base,omitempty"`
	Output     string            `yaml:"output,omitempty"` // "human" | "json"
	Shards     *bool             `yaml:"shards,omitempty"`
	Polymarket *PolymarketConfig `yaml:"polymarket,omitempty"`
}

// Dir returns the Supermodel config directory (~/.supermodel).
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".supermodel")
}

// Path returns the full path to the config file.
func Path() string {
	return filepath.Join(Dir(), "config.yaml")
}

// Load reads the config file. Returns defaults when the file does not exist.
// Environment variables override file values:
//   - SUPERMODEL_API_KEY overrides api_key
//   - SUPERMODEL_API_BASE overrides api_base
func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		cfg := defaults()
		cfg.applyEnv()
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	cfg.applyEnv()
	return &cfg, nil
}

// Save writes the config to disk, creating the directory if necessary.
// The file is written with mode 0600 (owner-readable only).
// Uses a tmp→rename pattern so a partial write (e.g. killed process) can
// never leave a corrupt config file.
func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	dest := Path()
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ShardsEnabled reports whether shard mode is on. Defaults to true.
func (c *Config) ShardsEnabled() bool {
	if c.Shards != nil {
		return *c.Shards
	}
	return true
}

// RequireAPIKey returns an actionable error if no API key is configured.
func (c *Config) RequireAPIKey() error {
	if c.APIKey == "" {
		return fmt.Errorf("not authenticated — run `supermodel login` first")
	}
	return nil
}

// EnsurePolymarket returns the Polymarket config sub-section, initializing
// it if it was nil (not yet present in the config file).
func (c *Config) EnsurePolymarket() *PolymarketConfig {
	if c.Polymarket == nil {
		c.Polymarket = &PolymarketConfig{}
	}
	return c.Polymarket
}

// RequirePolymarketKey returns an actionable error when no Polymarket credentials
// are configured.
func (c *Config) RequirePolymarketKey() error {
	if c.Polymarket == nil {
		return fmt.Errorf("Polymarket credentials not set — run `supermodel polymarket auth`")
	}
	if c.Polymarket.APIKey == "" && c.Polymarket.KeyID == "" {
		return fmt.Errorf("Polymarket credentials not set — run `supermodel polymarket auth` or set POLYMARKET_API_KEY / POLYMARKET_KEY_ID")
	}
	return nil
}

func defaults() *Config {
	return &Config{APIBase: DefaultAPIBase, Output: defaultOutput}
}

func (c *Config) applyDefaults() {
	if c.APIBase == "" {
		c.APIBase = DefaultAPIBase
	}
	if c.Output == "" {
		c.Output = defaultOutput
	}
}

func (c *Config) applyEnv() {
	if key := os.Getenv("SUPERMODEL_API_KEY"); key != "" {
		c.APIKey = key
	}
	if base := os.Getenv("SUPERMODEL_API_BASE"); base != "" {
		c.APIBase = base
	}
	if os.Getenv("SUPERMODEL_SHARDS") == "false" {
		c.Shards = boolPtr(false)
	}
	// CLOB API env overrides
	if key := os.Getenv("POLYMARKET_API_KEY"); key != "" {
		c.EnsurePolymarket().APIKey = key
	}
	if secret := os.Getenv("POLYMARKET_SECRET"); secret != "" {
		c.EnsurePolymarket().Secret = secret
	}
	if pass := os.Getenv("POLYMARKET_PASSPHRASE"); pass != "" {
		c.EnsurePolymarket().Passphrase = pass
	}
	// Builder API env overrides
	if kid := os.Getenv("POLYMARKET_KEY_ID"); kid != "" {
		c.EnsurePolymarket().KeyID = kid
	}
	if sk := os.Getenv("POLYMARKET_SECRET_KEY"); sk != "" {
		c.EnsurePolymarket().SecretKey = sk
	}
	// Wallet env overrides
	if pk := os.Getenv("POLYMARKET_PRIVATE_KEY"); pk != "" {
		c.EnsurePolymarket().PrivateKey = pk
	}
	if eoa := os.Getenv("POLYMARKET_EOA_ADDRESS"); eoa != "" {
		c.EnsurePolymarket().EOAAddress = eoa
	}
	if wallet := os.Getenv("POLYMARKET_PROXY_WALLET"); wallet != "" {
		c.EnsurePolymarket().ProxyWallet = wallet
	}
	if deposit := os.Getenv("POLYMARKET_DEPOSIT_WALLET"); deposit != "" {
		c.EnsurePolymarket().DepositWallet = deposit
	}
}

func boolPtr(b bool) *bool { return &b }
