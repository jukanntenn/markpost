package markpost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// Account endpoints: the caller's own sessions, post key, retention, password.

// GetRetention returns the caller's effective retention policy
// ({posts_days, history_days}).
func (c *Client) GetRetention(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, apiPrefix+"/me/retention", nil, nil, true)
}

// ListSessions returns the caller's refresh-token sessions.
func (c *Client) ListSessions(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, apiPrefix+"/auth/sessions", nil, nil, true)
}

// RevokeSession revokes one of the caller's sessions by token id.
func (c *Client) RevokeSession(ctx context.Context, tokenID int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodDelete, apiPrefix+"/auth/sessions/"+url.PathEscape(strconv.Itoa(tokenID)), nil, nil, true)
}

// RevokeOtherSessions revokes all of the caller's sessions except the current
// one (the backend keeps the requesting session alive).
func (c *Client) RevokeOtherSessions(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodDelete, apiPrefix+"/auth/sessions", nil, nil, true)
}

// RotatePostKey rotates the caller's post key; the old key stops working
// immediately.
func (c *Client) RotatePostKey(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, apiPrefix+"/post-key/rotate", nil, nil, true)
}

// ChangePassword sets a new password and adopts the fresh token pair the
// backend returns (sessions continue seamlessly).
func (c *Client) ChangePassword(ctx context.Context, currentPassword, newPassword string) (json.RawMessage, error) {
	raw, err := c.request(ctx, http.MethodPost, apiPrefix+"/auth/change-password",
		nil, changePasswordRequest{CurrentPassword: currentPassword, NewPassword: newPassword}, true)
	if err != nil {
		return nil, err
	}
	var res authResponse
	if err := json.Unmarshal(raw, &res); err == nil && res.Token != "" {
		c.adoptTokenPair(tokenPair{accessToken: res.Token, refreshToken: res.RefreshToken})
	}
	return raw, nil
}
