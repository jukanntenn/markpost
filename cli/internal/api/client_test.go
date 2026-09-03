package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/cli/internal/testserver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startFake mounts the shared fake backend and returns it with a client
// pointed at its URL.
func startFake(t *testing.T) (*testserver.Server, *Client) {
	t.Helper()
	fake := testserver.New()
	url := fake.Start(t)
	client := New(url, nil)
	client.SetUserAgent("markpost-cli/test")
	return fake, client
}

func TestLoginSuccess(t *testing.T) {
	_, client := startFake(t)
	login, err := client.Login(context.Background(), testserver.Username, testserver.Password)
	require.NoError(t, err)
	assert.Equal(t, testserver.Username, login.User.Username)
	assert.Equal(t, "at-1", login.Token)
	assert.Equal(t, "rt-1", login.RefreshToken)
	assert.Equal(t, int64(3600), login.ExpiresIn)
	// Login stores the session on the client for subsequent authed calls.
	assert.Equal(t, "at-1", client.AccessToken())
}

func TestLoginWrongPasswordIsPlainError(t *testing.T) {
	_, client := startFake(t)
	_, err := client.Login(context.Background(), testserver.Username, "nope")
	require.Error(t, err)
	assert.False(t, IsAuthError(err), "a bad password is not a session-expiry AuthError")
	httpErr, ok := AsHTTPError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
	assert.Equal(t, "invalid_credentials", httpErr.Code)
	assert.Equal(t, "invalid username or password", httpErr.Message)
}

func TestHTTPErrorMessage(t *testing.T) {
	e := &HTTPError{StatusCode: 422, Code: "validation_failed", Message: "request failed",
		FieldErrors: []FieldError{{Field: "title", Code: "required"}}}
	assert.Equal(t, "HTTP 422: validation_failed: request failed (title: required)", e.Error())
}

func TestAuthedCallWithoutTokenIsUnauthorized(t *testing.T) {
	_, client := startFake(t)
	_, err := client.ListPosts(context.Background(), ListPostsParams{})
	require.Error(t, err)
	httpErr, ok := AsHTTPError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
}

func TestRefreshOn401RetriesRequestAndFiresCallback(t *testing.T) {
	fake, client := startFake(t)
	// Start with a stale access token and a valid refresh token.
	client.SetSession("at-stale", "rt-1")
	var (
		gotAccess  string
		gotRefresh string
		gotExpires time.Time
	)
	client.SetTokensChanged(func(access, refresh string, expiresAt time.Time) {
		gotAccess, gotRefresh, gotExpires = access, refresh, expiresAt
	})

	list, err := client.ListPosts(context.Background(), ListPostsParams{})
	require.NoError(t, err)
	assert.Equal(t, "p-existing", list.Items[0].QID)

	assert.Equal(t, 1, fake.RefreshCalls(), "exactly one refresh per stale request")
	assert.Equal(t, "at-2", gotAccess)
	assert.Equal(t, "rt-2", gotRefresh)
	assert.False(t, gotExpires.IsZero())
	assert.Equal(t, "at-2", client.AccessToken())
}

func TestRefreshFailureBecomesAuthError(t *testing.T) {
	fake, client := startFake(t)
	client.SetSession("at-stale", "rt-revoked")
	_, err := client.ListPosts(context.Background(), ListPostsParams{})
	require.Error(t, err)
	assert.True(t, IsAuthError(err), "unrecoverable 401 must surface as AuthError, got %v", err)
	assert.Equal(t, 1, fake.RefreshCalls())
}

func TestNoRefreshTokenMeansNoRefreshAttempt(t *testing.T) {
	fake, client := startFake(t)
	client.SetSession("at-stale", "")
	_, err := client.ListPosts(context.Background(), ListPostsParams{})
	require.Error(t, err)
	assert.False(t, IsAuthError(err), "env-token sessions fail with the raw 401")
	assert.Equal(t, 0, fake.RefreshCalls())
}

func TestCreatePostByKeyAndFetch(t *testing.T) {
	_, client := startFake(t)
	qid, err := client.CreatePost(context.Background(), testserver.PostKey, "Hello", "World")
	require.NoError(t, err)
	assert.Equal(t, "p-2", qid)

	md, err := client.RawMarkdown(context.Background(), qid)
	require.NoError(t, err)
	assert.Equal(t, "# Hello\n\nWorld", md)

	html, err := client.PostHTML(context.Background(), qid)
	require.NoError(t, err)
	assert.Contains(t, html, "<h1>Hello</h1>")
}

func TestRawMarkdownNotFound(t *testing.T) {
	_, client := startFake(t)
	_, err := client.RawMarkdown(context.Background(), "p-missing")
	require.Error(t, err)
	httpErr, ok := AsHTTPError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
}

func TestDeletePost(t *testing.T) {
	fake, client := startFake(t)
	client.SetSession("at-1", "")
	require.NoError(t, client.DeletePost(context.Background(), "p-existing"))
	assert.False(t, fake.HasPost("p-existing"))
}

func TestReadyUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
	}))
	t.Cleanup(server.Close)
	client := New(server.URL, server.Client())
	status, err := client.Ready(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "unavailable", status)
}

func TestRetentionPostKeyRotate(t *testing.T) {
	_, client := startFake(t)
	client.SetSession("at-1", "")
	retention, err := client.Retention(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 7, retention.PostsDays)
	assert.Equal(t, 30, retention.HistoryDays)

	key, err := client.PostKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, testserver.PostKey, key.PostKey)

	rotated, err := client.RotatePostKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "mpk-rotated", rotated)
}

func TestHealthVersion(t *testing.T) {
	_, client := startFake(t)
	health, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", health)
	version, err := client.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v0.1.0", version)
}

func TestPassthrough(t *testing.T) {
	_, client := startFake(t)
	client.SetSession("at-1", "")
	status, body, _, err := client.Passthrough(context.Background(), PassthroughRequest{
		Method: http.MethodGet,
		Path:   ResolveAPIPath("me/retention"),
		Authed: true,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"posts_days":7,"history_days":30}`, string(body))

	status, _, _, err = client.Passthrough(context.Background(), PassthroughRequest{
		Method: http.MethodPost,
		Path:   "/" + testserver.PostKey,
		Body:   []byte(`{"title":"T","body":"B"}`),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, status)
}

func TestResolveAPIPath(t *testing.T) {
	assert.Equal(t, "/api/v1/me/retention", ResolveAPIPath("me/retention"))
	assert.Equal(t, "/api/v1/me/retention", ResolveAPIPath("/api/v1/me/retention"))
	assert.Equal(t, "/health", ResolveAPIPath("/health"))
	assert.Equal(t, "https://other/x", ResolveAPIPath("https://other/x"))
	assert.Equal(t, "", ResolveAPIPath(""))
}

func TestRequestCarriesUserAgentAndBearer(t *testing.T) {
	var gotUA, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"posts_days":1,"history_days":1}`))
	}))
	t.Cleanup(server.Close)
	client := New(server.URL, server.Client())
	client.SetUserAgent("markpost-cli/1.2.3 Agent/codex")
	client.SetSession("tok", "")
	_, err := client.Retention(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "markpost-cli/1.2.3 Agent/codex", gotUA)
	assert.Equal(t, "Bearer tok", gotAuth)
}
