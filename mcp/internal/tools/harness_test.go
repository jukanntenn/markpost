package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

// apiRoute is one canned backend response.
type apiRoute struct {
	Method string
	Path   string // may contain "?query" to assert exact query strings
	Status int    // default 200
	Body   string
	// Raw serves Body as-is (non-JSON responses like format=raw markdown).
	Raw bool
}

// fakeAPI is a canned markpost backend: login always works, every other
// route answers from the table (first match wins). Requests are recorded.
type fakeAPI struct {
	mu      sync.Mutex
	srv     *httptest.Server
	routes  []apiRoute
	request []string // "METHOD /path?query"
}

func newFakeAPI(t *testing.T, routes ...apiRoute) *fakeAPI {
	t.Helper()
	f := &fakeAPI{routes: routes}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	key := r.Method + " " + r.URL.Path
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	f.request = append(f.request, key)
	routes := f.routes
	f.mu.Unlock()

	if r.URL.Path == "/api/v1/auth/login" {
		writeJSON(w, http.StatusOK, `{"user":{"id":1,"username":"alice"},"token":"t1","refresh_token":"r1","expires_in":86400}`)
		return
	}
	for _, rt := range routes {
		pathMatch := strings.TrimSuffix(rt.Path, "/") == strings.TrimSuffix(r.URL.Path, "/") ||
			(rt.Method == r.Method && rt.Path == key)
		if rt.Method == r.Method && pathMatch {
			status := rt.Status
			if status == 0 {
				status = http.StatusOK
			}
			if rt.Raw {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(rt.Body))
				return
			}
			writeJSON(w, status, rt.Body)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, `{"error":{"code":"not_found","message":"no route"}}`)
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (f *fakeAPI) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.request...)
}

// harness wires an MCP server (via in-memory transports) to a fake backend.
type harness struct {
	fake    *fakeAPI
	session *mcp.ClientSession
}

func newHarness(t *testing.T, readOnly bool, toolsets []string, routes ...apiRoute) *harness {
	t.Helper()
	fake := newFakeAPI(t, routes...)
	client := markpost.NewClient(fake.srv.URL, "alice", "secret")
	require.NoError(t, client.Login(context.Background()))

	s := mcp.NewServer(&mcp.Implementation{Name: "markpost-mcp-test", Version: "test"}, nil)
	require.NoError(t, RegisterEnabled(s, client, toolsets, readOnly))

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := s.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	c := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := c.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return &harness{fake: fake, session: session}
}

// call invokes a tool and returns its text content joined.
func (h *harness) call(t *testing.T, name string, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	res, err := h.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err, "protocol-level failure calling %s", name)
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return res, sb.String()
}

// requireJSONEq asserts the tool text is equivalent JSON.
func requireJSONEq(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	require.NoError(t, json.Unmarshal([]byte(got), &g), "tool output is not JSON: %s", got)
	require.NoError(t, json.Unmarshal([]byte(want), &w))
	require.Equal(t, w, g, "tool output mismatch")
}
