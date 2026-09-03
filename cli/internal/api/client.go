// Package api is the markpost REST v1 client: typed methods over one
// request/retry core that injects the bearer token, transparently refreshes
// it once on a 401, and decodes the server's error envelope.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TokensChanged is invoked after a successful token refresh so the caller
// (the factory) can persist the new pair to the config file.
type TokensChanged func(accessToken, refreshToken string, expiresAt time.Time)

type Client struct {
	baseURL     string
	httpClient  *http.Client
	accessToken string
	// refreshToken is empty in token-from-env mode, where a 401 is terminal.
	refreshToken  string
	onTokenChange TokensChanged
	userAgent     string
	now           func() time.Time
}

func New(server string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL:    server,
		httpClient: httpClient,
		userAgent:  "markpost-cli",
		now:        time.Now,
	}
}

func (c *Client) SetUserAgent(ua string) { c.userAgent = ua }

// SetSession installs the tokens for authenticated calls.
func (c *Client) SetSession(accessToken, refreshToken string) {
	c.accessToken = accessToken
	c.refreshToken = refreshToken
}

func (c *Client) SetTokensChanged(cb TokensChanged) { c.onTokenChange = cb }

func (c *Client) AccessToken() string  { return c.accessToken }
func (c *Client) RefreshToken() string { return c.refreshToken }
func (c *Client) BaseURL() string      { return c.baseURL }

// request describes one API call. Body (JSON-encoded) or rawBody (verbatim
// bytes) is replayed verbatim on a post-refresh retry.
type request struct {
	method      string
	path        string
	query       url.Values
	body        any
	rawBody     []byte
	contentType string
	auth        bool
	// noRefresh skips the 401 auto-refresh: auth endpoints themselves must
	// surface a 401 as a plain error (a failed login is not a session
	// failure), and the refresh call must never recurse into itself.
	noRefresh bool
}

func (c *Client) do(ctx context.Context, r request) (result, error) {
	res, err := c.send(ctx, r)
	if err != nil {
		return res, err
	}
	if res.status == http.StatusUnauthorized && r.auth && !r.noRefresh && c.refreshToken != "" {
		if err := c.Refresh(ctx); err != nil {
			return result{}, &AuthError{Message: "session expired and could not be refreshed", Cause: err}
		}
		if res, err = c.send(ctx, r); err != nil {
			return result{}, err
		}
	}
	if res.status >= 400 {
		return res, newHTTPErrorStatus(res.status, res.body)
	}
	return res, nil
}

// result is the fully-drained outcome of one request. No *http.Response
// escapes this package: bodies are read and closed inside send, so callers
// never owe a Close.
type result struct {
	status int
	header http.Header
	body   []byte
}

func (c *Client) send(ctx context.Context, r request) (result, error) {
	u := c.baseURL + r.path
	if len(r.query) > 0 {
		u += "?" + r.query.Encode()
	}
	var reader io.Reader
	contentType := r.contentType
	if r.body != nil {
		encoded, err := json.Marshal(r.body)
		if err != nil {
			return result{}, fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
		contentType = "application/json"
	} else if r.rawBody != nil {
		reader = bytes.NewReader(r.rawBody)
		if contentType == "" {
			contentType = "application/json"
		}
	}
	req, err := http.NewRequestWithContext(ctx, r.method, u, reader)
	if err != nil {
		return result{}, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if r.auth && c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return result{}, fmt.Errorf("connect to %s: %w", c.baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return result{}, fmt.Errorf("read response: %w", err)
	}
	return result{status: resp.StatusCode, header: resp.Header, body: body}, nil
}

func decodeJSON(body []byte, v any) error {
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

// Login exchanges username/password for a session. The 401 a bad login
// returns is an ordinary error, not an AuthError.
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	res, err := c.do(ctx, request{
		method:    http.MethodPost,
		path:      "/api/v1/auth/login",
		body:      map[string]string{"username": username, "password": password},
		noRefresh: true,
	})
	if err != nil {
		return nil, err
	}
	var out LoginResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return nil, err
	}
	c.accessToken = out.Token
	c.refreshToken = out.RefreshToken
	return &out, nil
}

// Refresh exchanges the refresh token for a new pair, updates the session,
// and notifies the TokensChanged callback.
func (c *Client) Refresh(ctx context.Context) error {
	res, err := c.do(ctx, request{
		method:    http.MethodPost,
		path:      "/api/v1/auth/refresh",
		body:      map[string]string{"refresh_token": c.refreshToken},
		auth:      false,
		noRefresh: true,
	})
	if err != nil {
		return err
	}
	var out RefreshResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return err
	}
	c.accessToken = out.Token
	c.refreshToken = out.RefreshToken
	if c.onTokenChange != nil {
		c.onTokenChange(out.Token, out.RefreshToken, c.now().Add(time.Duration(out.ExpiresIn)*time.Second))
	}
	return nil
}

// Logout revokes the access token server-side. Callers treat errors as
// non-fatal: local state is cleared regardless.
func (c *Client) Logout(ctx context.Context) error {
	_, err := c.do(ctx, request{
		method: http.MethodPost,
		path:   "/api/v1/auth/logout",
		auth:   true,
	})
	return err
}

func (c *Client) PostKey(ctx context.Context) (*PostKeyResponse, error) {
	res, err := c.do(ctx, request{method: http.MethodGet, path: "/api/v1/post-key", auth: true})
	if err != nil {
		return nil, err
	}
	var out PostKeyResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RotatePostKey(ctx context.Context) (string, error) {
	res, err := c.do(ctx, request{method: http.MethodPost, path: "/api/v1/post-key/rotate", auth: true})
	if err != nil {
		return "", err
	}
	var out RotatePostKeyResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return "", err
	}
	return out.PostKey, nil
}

type ListPostsParams struct {
	Search string
	Page   int
	Limit  int
}

func (c *Client) ListPosts(ctx context.Context, p ListPostsParams) (*PostList, error) {
	q := url.Values{}
	if p.Search != "" {
		q.Set("search", p.Search)
	}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	res, err := c.do(ctx, request{method: http.MethodGet, path: "/api/v1/posts", query: q, auth: true})
	if err != nil {
		return nil, err
	}
	var out PostList
	if err := decodeJSON(res.body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePost(ctx context.Context, qid string) error {
	_, err := c.do(ctx, request{method: http.MethodDelete, path: "/api/v1/posts/" + url.PathEscape(qid), auth: true})
	return err
}

// CreatePost publishes via the post-key endpoint (POST /:post_key) — the only
// creation route the server offers — and returns the new post's QID.
func (c *Client) CreatePost(ctx context.Context, postKey, title, body string) (string, error) {
	res, err := c.do(ctx, request{
		method: http.MethodPost,
		path:   "/" + url.PathEscape(postKey),
		body:   map[string]string{"title": title, "body": body},
	})
	if err != nil {
		return "", err
	}
	var out CreatePostResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// RawMarkdown fetches the post as markdown text (public route; the server
// prefixes the title as a level-1 heading).
func (c *Client) RawMarkdown(ctx context.Context, qid string) (string, error) {
	return c.getText(ctx, "/"+url.PathEscape(qid)+"?format=raw")
}

// PostHTML fetches the rendered HTML page (public route).
func (c *Client) PostHTML(ctx context.Context, qid string) (string, error) {
	return c.getText(ctx, "/"+url.PathEscape(qid))
}

func (c *Client) getText(ctx context.Context, path string) (string, error) {
	res, err := c.do(ctx, request{method: http.MethodGet, path: path})
	if err != nil {
		return "", err
	}
	if len(res.body) == 0 {
		return "", fmt.Errorf("empty response from %s (content-type %s)", path, res.header.Get("Content-Type"))
	}
	return string(res.body), nil
}

func (c *Client) Retention(ctx context.Context) (*Retention, error) {
	res, err := c.do(ctx, request{method: http.MethodGet, path: "/api/v1/me/retention", auth: true})
	if err != nil {
		return nil, err
	}
	var out Retention
	if err := decodeJSON(res.body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Health(ctx context.Context) (string, error) {
	res, err := c.do(ctx, request{method: http.MethodGet, path: "/api/v1/health"})
	if err != nil {
		return "", err
	}
	var out HealthResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

func (c *Client) Ready(ctx context.Context) (string, error) {
	res, err := c.do(ctx, request{method: http.MethodGet, path: "/api/v1/ready"})
	if err != nil {
		// A 503 readiness probe carries the same {"status": "unavailable"}
		// envelope as the 200 case; do() surfaced it as an HTTPError.
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusServiceUnavailable {
			return "unavailable", nil
		}
		return "", err
	}
	var out HealthResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

func (c *Client) Version(ctx context.Context) (string, error) {
	res, err := c.do(ctx, request{method: http.MethodGet, path: "/api/v1/version"})
	if err != nil {
		return "", err
	}
	var out VersionResponse
	if err := decodeJSON(res.body, &out); err != nil {
		return "", err
	}
	return out.Version, nil
}

// PassthroughRequest carries a raw request for the `api` command. Body may be
// nil; ContentType defaults to application/json when a body is present.
type PassthroughRequest struct {
	Method      string
	Path        string
	Body        []byte
	ContentType string
	Authed      bool
}

// Passthrough executes an arbitrary request and returns status, body bytes,
// and the response Content-Type. It never auto-refreshes; the refresh dance
// belongs to the typed methods.
func (c *Client) Passthrough(ctx context.Context, pr PassthroughRequest) (int, []byte, string, error) {
	res, err := c.send(ctx, request{
		method:      pr.Method,
		path:        pr.Path,
		rawBody:     pr.Body,
		contentType: pr.ContentType,
		auth:        pr.Authed,
	})
	if err != nil {
		return 0, nil, "", err
	}
	return res.status, res.body, res.header.Get("Content-Type"), nil
}

// ResolveAPIPath applies the gh api convention: a path already starting with
// "/" or a full URL is used as-is; anything else is an endpoint relative to
// /api/v1.
func ResolveAPIPath(p string) string {
	if p == "" || p[0] == '/' || strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	return "/api/v1/" + p
}
