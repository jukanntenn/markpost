package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jukanntenn/markpost/mcp/internal/config"
)

func TestNewFailsFastOnBadCredentials(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_credentials","message":"bad credentials"}}`))
	}))
	t.Cleanup(fake.Close)

	cfg := &config.Config{
		BaseURL:  fake.URL,
		Username: "alice",
		Password: "wrong",
		Toolsets: []string{"posts"},
	}
	_, err := New(context.Background(), Options{Config: cfg})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_credentials")
}

func TestNewBuildsServerWithRequestedToolsets(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"id":1},"token":"t","refresh_token":"r","expires_in":1}`))
	}))
	t.Cleanup(fake.Close)

	cfg := &config.Config{
		BaseURL:  fake.URL,
		Username: "alice",
		Password: "secret",
		Toolsets: []string{"posts", "admin"},
		Version:  "vtest",
	}
	s, err := New(context.Background(), Options{Config: cfg})
	require.NoError(t, err)
	require.NotNil(t, s)

	// Drive the server in-memory to see the registered tools.
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := s.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })
	c := mcp.NewClient(&mcp.Implementation{Name: "t"}, nil)
	session, err := c.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, res.Tools, 32, "4 posts + 28 admin")
}

func TestHTTPHandlerBearerGuard(t *testing.T) {
	getServer := func(*http.Request) *mcp.Server {
		return mcp.NewServer(&mcp.Implementation{Name: "x", Version: "x"}, nil)
	}
	handler := HTTPHandler(getServer, "sesame")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Correct token reaches the MCP handler (which rejects the empty body
	// as a protocol error, not 401).
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer sesame")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestHTTPHandlerNoTokenOpen(t *testing.T) {
	getServer := func(*http.Request) *mcp.Server {
		return mcp.NewServer(&mcp.Implementation{Name: "x", Version: "x"}, nil)
	}
	handler := HTTPHandler(getServer, "")
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}
