// Package markpost is a thin REST client for a running markpost server. It
// mirrors the backend's /api/v1 and public-endpoint contracts (see
// backend/internal/api/rest/v1); endpoints that only forward JSON to MCP tool
// results are returned as raw bodies so field fidelity is never at risk.
//
// Authentication: markpost has no personal access tokens, so the client holds
// username/password credentials, logs in on demand, and keeps the session
// alive across the default 24h access-token expiry: a 401 triggers one
// refresh (markpost rotates refresh tokens, so the response pair replaces the
// stored pair), and a failed refresh falls back to a fresh login. Both retry
// paths run under the client mutex — concurrent refreshes with the same
// refresh token would trip the backend's token-theft reuse detection.
package markpost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client talks to one markpost instance as one user.
type Client struct {
	baseURL  string
	hc       *http.Client
	username string
	password string

	mu    sync.Mutex
	token tokenPair
}

type tokenPair struct {
	accessToken  string
	refreshToken string
}

const apiPrefix = "/api/v1"

// NewClient returns a client for the instance at baseURL (scheme://host[:port],
// no path) authenticating as username.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		hc:       &http.Client{Timeout: 30 * time.Second},
		username: username,
		password: password,
	}
}

// BaseURL exposes the instance URL (tools compose public post links from it).
func (c *Client) BaseURL() string { return c.baseURL }

// Login authenticates and stores the token pair. Called eagerly at startup
// for fail-fast and lazily again whenever the session cannot be recovered.
func (c *Client) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loginLocked(ctx)
}

func (c *Client) loginLocked(ctx context.Context) error {
	var res authResponse
	err := c.exchange(ctx, http.MethodPost, apiPrefix+"/auth/login",
		loginRequest{Username: c.username, Password: c.password}, "", &res)
	if err != nil {
		return fmt.Errorf("login as %q: %w", c.username, err)
	}
	c.token = tokenPair{accessToken: res.Token, refreshToken: res.RefreshToken}
	return nil
}

// accessToken returns a bearer token, logging in if this is the first call.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token.accessToken == "" {
		if err := c.loginLocked(ctx); err != nil {
			return "", err
		}
	}
	return c.token.accessToken, nil
}

// recoverSession is invoked after a 401: it refreshes (rotating the stored
// pair) or, when the refresh token is also rejected, logs in again. The failed
// token is compared under the lock so only one goroutine performs the
// recovery while others were in flight with the same stale token.
func (c *Client) recoverSession(ctx context.Context, failedAccessToken string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token.accessToken != failedAccessToken {
		// Another call already recovered the session; ride its token.
		return c.token.accessToken, nil
	}

	if c.token.refreshToken != "" {
		var res authResponse
		err := c.exchange(ctx, http.MethodPost, apiPrefix+"/auth/refresh",
			refreshRequest{RefreshToken: c.token.refreshToken}, "", &res)
		if err == nil {
			c.token = tokenPair{accessToken: res.Token, refreshToken: res.RefreshToken}
			return c.token.accessToken, nil
		}
	}
	if err := c.loginLocked(ctx); err != nil {
		return "", fmt.Errorf("session expired and recovery failed: %w", err)
	}
	return c.token.accessToken, nil
}

// adoptTokenPair replaces the stored pair after change-password (the backend
// issues a fresh pair so the client continues seamlessly).
func (c *Client) adoptTokenPair(t tokenPair) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = t
}

// request is one REST call. auth attaches the bearer token and enables the
// 401 recovery loop. JSON bodies in; body out as raw bytes (callers decide
// whether to decode or pass through).
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any, auth bool) ([]byte, error) {
	// doOnce fully reads and closes the body itself; only the status code
	// escapes so the 401 check needs no *http.Response.
	doOnce := func(bearer string) (out []byte, status int, err error) {
		var rdr io.Reader
		if body != nil {
			buf, err := json.Marshal(body)
			if err != nil {
				return nil, 0, err
			}
			rdr = bytes.NewReader(buf)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path+queryString(query), rdr)
		if err != nil {
			return nil, 0, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		res, err := c.hc.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer func() { _ = res.Body.Close() }()
		out, err = io.ReadAll(res.Body)
		if err != nil {
			return nil, res.StatusCode, err
		}
		if res.StatusCode >= 400 {
			return out, res.StatusCode, apiError(res.StatusCode, out)
		}
		return out, res.StatusCode, nil
	}

	if !auth {
		out, _, err := doOnce("")
		return out, err
	}

	bearer, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	out, status, err := doOnce(bearer)
	// A 401 arrives as an apiError from doOnce; it is not fatal — the
	// recovery loop below owns it. Anything else fails the call.
	if status != http.StatusUnauthorized {
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	bearer, err = c.recoverSession(ctx, bearer)
	if err != nil {
		return nil, err
	}
	out, _, err = doOnce(bearer)
	return out, err
}

// exchange performs a JSON call and decodes the response into out.
func (c *Client) exchange(ctx context.Context, method, path string, body any, bearer string, out any) error {
	full := c.baseURL + path
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, full, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return apiError(res.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func queryString(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}
