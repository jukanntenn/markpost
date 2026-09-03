package markpost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordedRequest captures one call hitting the fake backend.
type recordedRequest struct {
	Method string
	Path   string // path only, query stripped
	Query  string
	Auth   string
	Body   string
}

// fakeBackend is a minimal markpost REST double backed by Go 1.22 method
// routing. Handlers are registered per test; every request is recorded.
type fakeBackend struct {
	t       *testing.T
	mu      sync.Mutex
	srv     *httptest.Server
	muxVal  *http.ServeMux
	counter atomic.Int64

	requests []recordedRequest
	// accessValid/refreshValid track which tokens count as live; the
	// 401/recovery tests flip them to simulate expiry.
	accessValid   map[string]bool
	refreshValid  map[string]bool
	currentAccess string
}

func newFakeBackend(t *testing.T) *fakeBackend {
	t.Helper()
	f := &fakeBackend{
		t:            t,
		muxVal:       http.NewServeMux(),
		accessValid:  map[string]bool{},
		refreshValid: map[string]bool{},
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.recordAndServe))
	t.Cleanup(f.srv.Close)
	return f
}

// handle registers a method+pattern handler (Go 1.22 ServeMux syntax).
func (f *fakeBackend) handle(pattern string, h http.HandlerFunc) {
	f.muxVal.HandleFunc(pattern, h)
}

func (f *fakeBackend) recordAndServe(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body)) // downstream handlers re-read it
	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
		Auth:   r.Header.Get("Authorization"),
		Body:   string(body),
	})
	valid := f.accessValid[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	f.mu.Unlock()

	// The login/refresh endpoints must work regardless of bearer state.
	switch r.URL.Path {
	case "/api/v1/auth/login", "/api/v1/auth/refresh":
		// fall through to the mux
	default:
		if r.Header.Get("Authorization") != "" && !valid {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": "invalid_token", "message": "Invalid or expired token"},
			})
			return
		}
	}
	f.muxVal.ServeHTTP(w, r)
}

// issueTokenPair simulates the backend issuing a rotated pair.
func (f *fakeBackend) issueTokenPair() (access, refresh string) {
	access = "acc-" + strconv.FormatInt(f.counter.Add(1), 10)
	refresh = "ref-" + strconv.FormatInt(f.counter.Add(1), 10)
	f.mu.Lock()
	f.accessValid[access] = true
	f.refreshValid[refresh] = true
	f.currentAccess = access
	f.mu.Unlock()
	return access, refresh
}

func (f *fakeBackend) recorded() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedRequest(nil), f.requests...)
}

// loginHandler answers POST /api/v1/auth/login with a fresh pair.
func (f *fakeBackend) loginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		access, refresh := f.issueTokenPair()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":          map[string]any{"id": 1, "username": "alice"},
			"token":         access,
			"refresh_token": refresh,
			"expires_in":    86400,
		})
	}
}

// refreshHandler answers POST /api/v1/auth/refresh when the presented
// refresh token is still valid (rotation: both sides become single-use).
func (f *fakeBackend) refreshHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		valid := f.refreshValid[req.RefreshToken]
		delete(f.refreshValid, req.RefreshToken)
		f.mu.Unlock()
		if !valid {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": "invalid_token", "message": "Invalid or expired token"},
			})
			return
		}
		access, refresh := f.issueTokenPair()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":         access,
			"refresh_token": refresh,
			"expires_in":    86400,
		})
	}
}

func TestClient_LoginStoresTokensAndAuthenticatesRequests(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	var access string
	f.handle("GET /api/v1/me/retention", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer "+access, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"posts_days":7,"history_days":14}`))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	f.mu.Lock()
	access = f.currentAccess
	f.mu.Unlock()

	raw, err := c.GetRetention(context.Background())
	require.NoError(t, err)
	assert.JSONEq(t, `{"posts_days":7,"history_days":14}`, string(raw))
	assert.Equal(t, "POST /api/v1/auth/login", f.recorded()[0].Method+" "+f.recorded()[0].Path)
}

func TestClient_RefreshesOn401AndRetries(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	f.handle("POST /api/v1/auth/refresh", f.refreshHandler())
	f.handle("GET /api/v1/me/retention", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"posts_days":0,"history_days":0}`))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	// Expire the access token server-side: the next call 401s and must
	// transparently refresh and retry.
	f.mu.Lock()
	f.accessValid = map[string]bool{}
	f.mu.Unlock()

	raw, err := c.GetRetention(context.Background())
	require.NoError(t, err)
	assert.JSONEq(t, `{"posts_days":0,"history_days":0}`, string(raw))

	var paths []string
	for _, req := range f.recorded() {
		paths = append(paths, req.Method+" "+req.Path)
	}
	assert.Equal(t, []string{
		"POST /api/v1/auth/login",
		"GET /api/v1/me/retention",
		"POST /api/v1/auth/refresh",
		"GET /api/v1/me/retention",
	}, paths)
}

func TestClient_ReLogsInWhenRefreshIsRejected(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	f.handle("POST /api/v1/auth/refresh", f.refreshHandler())
	f.handle("GET /api/v1/me/retention", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"posts_days":1,"history_days":1}`))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	// Kill both sides of the pair: refresh 401s, so the client must fall
	// back to a full login.
	f.mu.Lock()
	f.accessValid = map[string]bool{}
	f.refreshValid = map[string]bool{}
	f.mu.Unlock()

	raw, err := c.GetRetention(context.Background())
	require.NoError(t, err)
	assert.JSONEq(t, `{"posts_days":1,"history_days":1}`, string(raw))

	var paths []string
	for _, req := range f.recorded() {
		paths = append(paths, req.Method+" "+req.Path)
	}
	assert.Equal(t, []string{
		"POST /api/v1/auth/login",
		"GET /api/v1/me/retention",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/login",
		"GET /api/v1/me/retention",
	}, paths)
}

func TestClient_CreatePostFetchesKeyThenPostsPublicly(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	f.handle("GET /api/v1/post-key", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"post_key":"pk-123","created_at":"2026-09-03T00:00:00Z"}`))
	})
	f.handle("POST /pk-123", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.Header.Get("Authorization"), "public endpoint must not carry a bearer")
		var req createPostRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "Hello", req.Title)
		assert.Equal(t, "**world**", req.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"ab12cd"}`))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	res, err := c.CreatePost(context.Background(), "Hello", "**world**")
	require.NoError(t, err)
	assert.Equal(t, "ab12cd", res.ID)
	assert.Equal(t, f.srv.URL+"/ab12cd", res.URL)
}

func TestClient_GetPostRawUsesFormatRaw(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	f.handle("GET /ab12cd", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "format=raw", r.URL.RawQuery)
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = w.Write([]byte("# Hello\n\n**world**"))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	raw, err := c.GetPostRaw(context.Background(), "ab12cd")
	require.NoError(t, err)
	assert.Equal(t, "# Hello\n\n**world**", string(raw))
}

func TestClient_ListPostsBuildsQuery(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	f.handle("GET /api/v1/posts", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "limit=50&page=2&search=hello", r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":2,"limit":50,"total_pages":0}`))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	raw, err := c.ListPosts(context.Background(), "hello", 2, 50)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"total":0`)
}

func TestClient_APIErrorMapping(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", f.loginHandler())
	f.handle("DELETE /api/v1/posts/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"post not found"}}`))
	})

	c := NewClient(f.srv.URL, "alice", "secret")
	require.NoError(t, c.Login(context.Background()))

	err := c.DeletePost(context.Background(), "nope")
	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr), "want *APIError, got %T", err)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "not_found", apiErr.Code)
	assert.Equal(t, "post not found", apiErr.Message)
}

func TestClient_LoginFailureCarriesCredentials(t *testing.T) {
	f := newFakeBackend(t)
	f.handle("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_credentials","message":"bad username or password"}}`))
	})

	c := NewClient(f.srv.URL, "alice", "wrong")
	err := c.Login(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `login as "alice"`)
	assert.Contains(t, err.Error(), "invalid_credentials")
}
