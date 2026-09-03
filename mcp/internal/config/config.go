// Package config resolves markpost-mcp configuration from flags and the
// MARKPOST_MCP_* environment. Credentials are environment-only by design —
// never flags — so they stay out of shell history (the same trust model the
// golden reference applies to its PAT).
package config

import (
	"fmt"
	"os"
	"strings"
)

// DefaultToolsets are enabled unless MARKPOST_MCP_TOOLSETS/--toolsets says
// otherwise. The admin toolset is opt-in: most credentials are not admin and
// its tools are the most destructive surface.
const DefaultToolsets = "posts,delivery,account"

// Config is the resolved runtime configuration.
type Config struct {
	// BaseURL is the markpost instance URL (scheme://host[:port]).
	BaseURL string
	// Username/Password authenticate against the instance's /auth/login.
	Username string
	Password string
	// Toolsets lists enabled toolset names ("all" expands to every toolset).
	Toolsets []string
	// ReadOnly disables every tool that writes.
	ReadOnly bool
	// HTTPAddr/HTTPPath/HTTPToken configure the http subcommand; HTTPToken,
	// when set, is the bearer clients must present to the MCP endpoint.
	HTTPAddr  string
	HTTPPath  string
	HTTPToken string
	// Version is injected at build time.
	Version string
}

// Credentials returns an error naming the missing environment variables.
func (c *Config) Credentials() error {
	var missing []string
	if c.Username == "" {
		missing = append(missing, "MARKPOST_MCP_USERNAME")
	}
	if c.Password == "" {
		missing = append(missing, "MARKPOST_MCP_PASSWORD")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing credentials: set %s (environment-only by design)", strings.Join(missing, ", "))
	}
	return nil
}

// CredentialsFromEnv reads the environment-only credential variables.
func CredentialsFromEnv() (username, password string) {
	return os.Getenv("MARKPOST_MCP_USERNAME"), os.Getenv("MARKPOST_MCP_PASSWORD")
}

// ParseToolsets splits and normalizes a comma-separated toolset list.
func ParseToolsets(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
