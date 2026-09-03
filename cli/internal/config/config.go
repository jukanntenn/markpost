// Package config owns the CLI's on-disk state: one TOML file under the XDG
// config directory holding the chosen server and the login session (gh keeps
// the same split in config.yml/hosts.yml; a single file suffices at
// markpost's single-server scale). Environment variables override the file —
// resolution order lives in this package so every command agrees.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Auth is the persisted session block. Tokens are stored plaintext like gh's
// hosts.yml fallback; the file is created 0600 in a 0700 directory.
type Auth struct {
	Server       string `toml:"server"`
	UserID       int    `toml:"user_id"`
	Username     string `toml:"username"`
	Name         string `toml:"name"`
	Role         string `toml:"role"`
	Token        string `toml:"token"`
	RefreshToken string `toml:"refresh_token"`
	// ExpiresAt is the access token's expiry as unix seconds; zero when the
	// expiry is unknown (e.g. a --token login without refresh).
	ExpiresAt int64 `toml:"expires_at"`
}

type Config struct {
	Auth Auth `toml:"auth"`
}

// FileName is the config file's base name inside ConfigDir().
const FileName = "config.toml"

// DefaultPath is where the CLI reads and writes state in a real invocation:
// $MARKPOST_CONFIG_DIR, else $XDG_CONFIG_HOME/markpost, else ~/.config/markpost.
func DefaultPath() (string, error) {
	if dir := os.Getenv("MARKPOST_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, FileName), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "markpost", FileName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "markpost", FileName), nil
}

// Load reads the config file. A missing file is not an error — it is a fresh
// machine, returned as an empty Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// Save atomically replaces the config file (write to a sibling temp file,
// then rename), creating the directory 0700 and the file 0600 because it
// carries bearer and refresh tokens.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	// Best-effort cleanup: after a successful rename the path no longer
	// exists and the remove is a no-op that must not mask the real result.
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Rename(tmp.Name(), path)
}

// NormalizeServer validates and canonicalizes a user-supplied server URL:
// scheme must be http(s), host required, no trailing slash, no path or query.
func NormalizeServer(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return "", errors.New("empty server URL")
	}
	switch {
	case strings.HasPrefix(s, "http://"), strings.HasPrefix(s, "https://"):
	default:
		return "", fmt.Errorf("server URL must start with http:// or https://: %q", raw)
	}
	// "scheme://host" splits into exactly 3 parts; anything beyond is a path.
	if strings.ContainsAny(s, "?#") || len(strings.Split(s, "/")) > 3 {
		return "", fmt.Errorf("server URL must not contain a path, query, or fragment: %q", raw)
	}
	return s, nil
}

// ResolveServer applies MARKPOST_SERVER over the stored server, then
// normalizes. The --server flag (which folds in the same env var via its
// EnvVars) is applied by the caller before this — the factory owns flag
// precedence, this owns environment precedence.
func ResolveServer(cfg *Config) (string, error) {
	s := os.Getenv("MARKPOST_SERVER")
	if s == "" {
		s = cfg.Auth.Server
	}
	if s == "" {
		return "", ErrNoServer
	}
	return NormalizeServer(s)
}

// ResolveToken applies MARKPOST_TOKEN over the stored access token. fromEnv
// reports an environment token: such a session has no refresh token, so a 401
// is terminal rather than refreshable. The stored token is only offered for
// the server it was issued by — a different resolved server invalidates it.
func ResolveToken(cfg *Config, server string) (token string, fromEnv bool) {
	if t := os.Getenv("MARKPOST_TOKEN"); t != "" {
		return t, true
	}
	if cfg.Auth.Token != "" && cfg.Auth.Server == server {
		return cfg.Auth.Token, false
	}
	return "", false
}

// ErrNoServer means neither MARKPOST_SERVER nor a stored server exists.
var ErrNoServer = errors.New("no server configured: run 'markpost config set server <url>' or pass --server")
