// Package server assembles the markpost MCP server: one REST client, the
// enabled toolsets, and the server identity. Both transports (stdio, http)
// start from here so they always expose the same tool surface.
package server

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jukanntenn/markpost/mcp/internal/config"
	"github.com/jukanntenn/markpost/mcp/internal/markpost"
	"github.com/jukanntenn/markpost/mcp/internal/tools"
)

// Options carries the resolved configuration plus injectables for tests.
type Options struct {
	Config *config.Config
	// Client overrides the default REST client (tests inject one pointed at
	// an httptest server).
	Client *markpost.Client
	// SkipLogin disables the eager login (tests exercise login separately
	// with their own fakes).
	SkipLogin bool
}

// New builds the MCP server. It logs in eagerly so a bad URL or credential
// fails at startup instead of at the first tool call.
func New(ctx context.Context, opts Options) (*mcp.Server, error) {
	cfg := opts.Config
	client := opts.Client
	if client == nil {
		client = markpost.NewClient(cfg.BaseURL, cfg.Username, cfg.Password)
	}
	if !opts.SkipLogin {
		if err := client.Login(ctx); err != nil {
			return nil, err
		}
	}

	names := cfg.Toolsets
	if len(names) == 1 && names[0] == "all" {
		names = tools.AllNames()
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "markpost-mcp",
		Title:   "Markpost MCP Server",
		Version: orDefault(cfg.Version, "dev"),
	}, nil)
	if err := tools.RegisterEnabled(s, client, names, cfg.ReadOnly); err != nil {
		return nil, err
	}
	return s, nil
}

// HTTPHandler serves the MCP streamable-http endpoint (stateless: one server
// instance per connection-free request, the mode the golden reference uses
// for its remote server). When token is set, requests must present it as a
// bearer token.
func HTTPHandler(getServer func(*http.Request) *mcp.Server, token string) http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})
	if token == "" {
		return mcpHandler
	}
	return bearerGuard(token, mcpHandler)
}

// bearerGuard rejects requests without the expected bearer token, comparing
// in constant time.
func bearerGuard(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
