// Package tools defines markpost-mcp's toolsets. Each toolset mirrors one
// area of the markpost REST API (backend/internal/api/rest/v1) and consists
// of a set of tools registered with typed handlers — the go-sdk derives each
// tool's InputSchema from the handler's args struct. Handlers return the
// backend's JSON verbatim as text content; error paths surface the backend's
// own code and message.
package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

// Toolset is one named, toggleable group of tools.
type Toolset struct {
	// Default reports whether the toolset is enabled without explicit
	// --toolsets configuration.
	Default bool
	// Register adds the toolset's tools to s. readOnly registration skips
	// write tools entirely so read-only mode is enforced server-side.
	Register func(s *mcp.Server, c *markpost.Client, readOnly bool)
}

// Registry lists all toolsets in stable order. Names are part of the public
// CLI surface (--toolsets) and the snapshot tests.
var Registry = []struct {
	Name    string
	Toolset Toolset
}{
	{"posts", Toolset{Default: true, Register: registerPosts}},
	{"delivery", Toolset{Default: true, Register: registerDelivery}},
	{"account", Toolset{Default: true, Register: registerAccount}},
	{"admin", Toolset{Default: false, Register: registerAdmin}},
}

// IsToolset reports whether name is a known toolset.
func IsToolset(name string) bool {
	for _, r := range Registry {
		if r.Name == name {
			return true
		}
	}
	return false
}

// DefaultToolsetNames returns the default-enabled toolset names.
func DefaultToolsetNames() []string {
	var out []string
	for _, r := range Registry {
		if r.Toolset.Default {
			out = append(out, r.Name)
		}
	}
	return out
}

// AllNames returns every toolset name in registry order.
func AllNames() []string {
	out := make([]string, len(Registry))
	for i, r := range Registry {
		out[i] = r.Name
	}
	return out
}

// RegisterEnabled adds the named toolsets to s (read-only mode drops write
// tools). All names are validated before anything is registered — an unknown
// name is an error so typos fail loudly at startup without leaving a
// partially populated server.
func RegisterEnabled(s *mcp.Server, c *markpost.Client, names []string, readOnly bool) error {
	byName := make(map[string]Toolset, len(Registry))
	for _, r := range Registry {
		byName[r.Name] = r.Toolset
	}
	for _, name := range names {
		if _, ok := byName[name]; !ok {
			return fmt.Errorf("unknown toolset %q (known: %s)", name, allToolsetNames())
		}
	}
	for _, name := range names {
		byName[name].Register(s, c, readOnly)
	}
	return nil
}

func allToolsetNames() string {
	names := make([]string, len(Registry))
	for i, r := range Registry {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

// pageArgs is the shared pagination input. The backend caps limit at 100 and
// defaults to page 1 / limit 20 when the fields are zero.
type pageArgs struct {
	Page  int `json:"page,omitempty" jsonschema:"page number, 1-based (default 1)"`
	Limit int `json:"limit,omitempty" jsonschema:"items per page, at most 100 (default 20)"`
}

// rawResult emits raw REST JSON as indented text content.
func rawResult(raw json.RawMessage) (*mcp.CallToolResult, any, error) {
	return textResult(string(indentJSON(raw))), nil, nil
}

// textResult wraps s as the tool's text content.
func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// errorResult turns a markpost/markpost-client failure into a tool error
// carrying the backend's own code and message (protocol-level errors are
// reserved for MCP infrastructure failures).
func errorResult(action string, err error) (*mcp.CallToolResult, any, error) {
	msg := fmt.Sprintf("%s failed: %v", action, err)
	var e *markpost.APIError
	if errors.As(err, &e) {
		msg = fmt.Sprintf("%s failed: %s (HTTP %d)", action, e.Message, e.StatusCode)
		if len(e.FieldErrors) > 0 {
			details, _ := json.Marshal(e.FieldErrors)
			msg += " " + string(details)
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}, nil, nil
}

func indentJSON(raw json.RawMessage) []byte {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return raw
	}
	return buf.Bytes()
}
