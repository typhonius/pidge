package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type GatewayConfig struct {
	URL      string `toml:"url"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type ServerConfig struct {
	Listen        string `toml:"listen"`
	DBPath        string `toml:"db_path"`
	WebhookSecret string `toml:"webhook_secret"`
	AutoRegister  bool   `toml:"auto_register"`
	WebhookURL    string `toml:"webhook_url"`
	TLSCert       string `toml:"tls_cert"`
	TLSKey        string `toml:"tls_key"`
}

type Config struct {
	Gateway GatewayConfig `toml:"gateway"`
	Server  ServerConfig  `toml:"server"`
}

// DefaultPath returns ~/.config/pidge/config.toml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pidge", "config.toml"), nil
}

// DefaultDBPath returns ~/.config/pidge/pidge.db.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "pidge", "pidge.db"), nil
}

// Load reads the config from path and merges env var overrides.
func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cfg.applyDefaults()
	cfg.applyEnv()
	return &cfg, nil
}

// applyDefaults fills in zero-value fields with sensible defaults.
func (c *Config) applyDefaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = ":3851"
	}
	if c.Server.DBPath == "" {
		if p, err := DefaultDBPath(); err == nil {
			c.Server.DBPath = p
		}
	}
}

// applyEnv overrides config values with environment variables if set.
func (c *Config) applyEnv() {
	if v := os.Getenv("PIDGE_URL"); v != "" {
		c.Gateway.URL = v
	}
	if v := os.Getenv("PIDGE_USER"); v != "" {
		c.Gateway.Username = v
	}
	if v := os.Getenv("PIDGE_PASS"); v != "" {
		c.Gateway.Password = v
	}
	if v := os.Getenv("PIDGE_LISTEN"); v != "" {
		c.Server.Listen = v
	}
	if v := os.Getenv("PIDGE_DB_PATH"); v != "" {
		c.Server.DBPath = v
	}
	if v := os.Getenv("PIDGE_WEBHOOK_SECRET"); v != "" {
		c.Server.WebhookSecret = v
	}
}

// Validate checks that required fields are present.
func (c *Config) Validate() error {
	if c.Gateway.URL == "" {
		return fmt.Errorf("gateway URL is required")
	}
	if c.Gateway.Username == "" {
		return fmt.Errorf("gateway username is required")
	}
	if c.Gateway.Password == "" {
		return fmt.Errorf("gateway password is required")
	}
	return nil
}

// ExpandDBPath resolves ~ in the DB path to the user's home directory.
func (c *Config) ExpandDBPath() string {
	p := c.Server.DBPath
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	return p
}

// Save writes the config to the given path, creating directories as needed.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}
