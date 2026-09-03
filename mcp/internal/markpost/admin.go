package markpost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// Admin endpoints. All require an admin JWT; the client's user must hold the
// admin role or every call returns 403.

func adminPath(segments ...string) string {
	p := apiPrefix + "/admin"
	for _, s := range segments {
		p += "/" + s
	}
	return p
}

// ListUsers returns all users (paginated, username LIKE search).
func (c *Client) ListUsers(ctx context.Context, search string, page, limit int) (json.RawMessage, error) {
	q := paginationQuery(page, limit)
	if search != "" {
		q.Set("search", search)
	}
	return c.request(ctx, http.MethodGet, adminPath("users"), q, nil, true)
}

// GetUser returns one user's admin profile.
func (c *Client) GetUser(ctx context.Context, id int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("users", strconv.Itoa(id)), nil, nil, true)
}

// CreateUserInput mirrors the admin create-user body (password length policy
// is enforced server-side).
type CreateUserInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUser creates a user (201, AdminUserItem).
func (c *Client) CreateUser(ctx context.Context, in CreateUserInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, adminPath("users"), nil, in, true)
}

// SetUserRole sets role to "admin" or "user".
func (c *Client) SetUserRole(ctx context.Context, id int, role string) (json.RawMessage, error) {
	body := map[string]string{"role": role}
	return c.request(ctx, http.MethodPatch, adminPath("users", strconv.Itoa(id), "role"), nil, body, true)
}

// DeleteUser removes a user and their data; the response reports how many
// rows were deleted.
func (c *Client) DeleteUser(ctx context.Context, id int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodDelete, adminPath("users", strconv.Itoa(id)), nil, nil, true)
}

// ResetUserPassword generates a temporary password returned exactly once.
func (c *Client) ResetUserPassword(ctx context.Context, id int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, adminPath("users", strconv.Itoa(id), "password"), nil, nil, true)
}

// SetUserActive activates or deactivates a user.
func (c *Client) SetUserActive(ctx context.Context, id int, active bool) (json.RawMessage, error) {
	body := map[string]bool{"active": active}
	return c.request(ctx, http.MethodPatch, adminPath("users", strconv.Itoa(id), "active"), nil, body, true)
}

// SetUserVIP sets a user's VIP status.
func (c *Client) SetUserVIP(ctx context.Context, id int, vip bool) (json.RawMessage, error) {
	body := map[string]bool{"vip": vip}
	return c.request(ctx, http.MethodPatch, adminPath("users", strconv.Itoa(id), "vip"), nil, body, true)
}

// SetUserRetention sets one user's retention policy: nil inherits the global
// default, 0 keeps forever, 1-3650 keeps N days.
func (c *Client) SetUserRetention(ctx context.Context, id int, retentionDays *int) (json.RawMessage, error) {
	body := map[string]*int{"retention_days": retentionDays}
	return c.request(ctx, http.MethodPatch, adminPath("users", strconv.Itoa(id), "retention"), nil, body, true)
}

// RetentionTargetInput is shared by bulk-set and impact-preview: explicit
// user ids (max 200) or scope "vip", plus the candidate policy.
type RetentionTargetInput struct {
	UserIDs       []int  `json:"user_ids"`
	Scope         string `json:"scope"`
	RetentionDays *int   `json:"retention_days"`
}

// BulkSetRetention applies one retention policy to explicit users or all VIP users.
func (c *Client) BulkSetRetention(ctx context.Context, in RetentionTargetInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, adminPath("users", "retention", "bulk"), nil, in, true)
}

// RetentionImpact previews (without writing) the deletion a candidate policy
// would cause.
func (c *Client) RetentionImpact(ctx context.Context, in RetentionTargetInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, adminPath("retention", "impact"), nil, in, true)
}

// RetentionDefaults reports the global retention fallback windows.
func (c *Client) RetentionDefaults(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("retention", "defaults"), nil, nil, true)
}

// ListUserSessions lists a user's sessions (admin view).
func (c *Client) ListUserSessions(ctx context.Context, userID int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("users", strconv.Itoa(userID), "sessions"), nil, nil, true)
}

// RevokeUserSessions revokes every session of a user.
func (c *Client) RevokeUserSessions(ctx context.Context, userID int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodDelete, adminPath("users", strconv.Itoa(userID), "sessions"), nil, nil, true)
}

// RevokeSessionByID revokes a single session by token id.
func (c *Client) RevokeSessionByID(ctx context.Context, tokenID int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodDelete, adminPath("sessions", strconv.Itoa(tokenID)), nil, nil, true)
}

// ListAllPosts returns all posts (paginated, optional search and username filter).
func (c *Client) ListAllPosts(ctx context.Context, search, username string, page, limit int) (json.RawMessage, error) {
	q := paginationQuery(page, limit)
	if search != "" {
		q.Set("search", search)
	}
	if username != "" {
		q.Set("username", username)
	}
	return c.request(ctx, http.MethodGet, adminPath("posts"), q, nil, true)
}

// DeleteAnyPost removes any post by QID (204).
func (c *Client) DeleteAnyPost(ctx context.Context, qid string) error {
	_, err := c.request(ctx, http.MethodDelete, adminPath("posts", url.PathEscape(qid)), nil, nil, true)
	return err
}

// ListAllChannels returns all delivery channels across users (paginated).
func (c *Client) ListAllChannels(ctx context.Context, page, limit int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("delivery", "channels"), paginationQuery(page, limit), nil, true)
}

// AdminCreateChannelInput mirrors the admin create-channel body.
type AdminCreateChannelInput struct {
	UserID        int             `json:"user_id"`
	Kind          string          `json:"kind"`
	Name          string          `json:"name"`
	Configuration json.RawMessage `json:"configuration"`
	Keywords      string          `json:"keywords"`
}

// CreateChannelForUser creates a channel owned by the given user (201).
func (c *Client) CreateChannelForUser(ctx context.Context, in AdminCreateChannelInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, adminPath("delivery", "channels"), nil, in, true)
}

// SetChannelEnabled enables or disables any channel (admin).
func (c *Client) SetChannelEnabled(ctx context.Context, id int, enabled bool) (json.RawMessage, error) {
	body := map[string]bool{"enabled": enabled}
	return c.request(ctx, http.MethodPatch, adminPath("delivery", "channels", strconv.Itoa(id), "enabled"), nil, body, true)
}

// DeleteChannelByID removes any channel (admin).
func (c *Client) DeleteChannelByID(ctx context.Context, id int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodDelete, adminPath("delivery", "channels", strconv.Itoa(id)), nil, nil, true)
}

// AdminHistoryFilter mirrors the admin delivery-history query filters.
type AdminHistoryFilter struct {
	UserID        int
	ChannelID     int
	Status        string
	ErrorCategory string
}

// ListAllDeliveryHistory returns delivery history across users (paginated,
// filtered). ErrorCategory must be one of card_rejected,
// upstream_client_error, upstream_server_error, upstream_business_error,
// network, internal.
func (c *Client) ListAllDeliveryHistory(ctx context.Context, f AdminHistoryFilter, page, limit int) (json.RawMessage, error) {
	q := paginationQuery(page, limit)
	if f.UserID > 0 {
		q.Set("user_id", strconv.Itoa(f.UserID))
	}
	if f.ChannelID > 0 {
		q.Set("channel_id", strconv.Itoa(f.ChannelID))
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.ErrorCategory != "" {
		q.Set("error_category", f.ErrorCategory)
	}
	return c.request(ctx, http.MethodGet, adminPath("delivery", "history"), q, nil, true)
}

// GetSiteDeliveryStats returns the site-wide delivery trend (days ≤ 365).
func (c *Client) GetSiteDeliveryStats(ctx context.Context, days int) (json.RawMessage, error) {
	q := url.Values{}
	if days > 0 {
		q.Set("days", strconv.Itoa(days))
	}
	return c.request(ctx, http.MethodGet, adminPath("delivery", "stats"), q, nil, true)
}

// ListLockedChannels lists channels flagged for persistent delivery failures.
func (c *Client) ListLockedChannels(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("locked-channels"), nil, nil, true)
}

// AuditLogQuery mirrors the admin audit-log filters (Since/Until are RFC3339).
type AuditLogQuery struct {
	ActorID    int
	Action     string
	TargetType string
	TargetID   string
	Since      string
	Until      string
}

// ListAuditLogs returns audit log entries with action-count facets.
func (c *Client) ListAuditLogs(ctx context.Context, qy AuditLogQuery, page, limit int) (json.RawMessage, error) {
	q := paginationQuery(page, limit)
	if qy.ActorID > 0 {
		q.Set("actor_id", strconv.Itoa(qy.ActorID))
	}
	if qy.Action != "" {
		q.Set("action", qy.Action)
	}
	if qy.TargetType != "" {
		q.Set("target_type", qy.TargetType)
	}
	if qy.TargetID != "" {
		q.Set("target_id", qy.TargetID)
	}
	if qy.Since != "" {
		q.Set("since", qy.Since)
	}
	if qy.Until != "" {
		q.Set("until", qy.Until)
	}
	return c.request(ctx, http.MethodGet, adminPath("audit-logs"), q, nil, true)
}

// GetStats returns the admin dashboard counters with week deltas.
func (c *Client) GetStats(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("stats"), nil, nil, true)
}

// GetSettings lists the runtime settings rows.
func (c *Client) GetSettings(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, adminPath("settings"), nil, nil, true)
}

// SetSettingInput mirrors the setting upsert body: the "vip" key owns
// {"enabled"}, "vip_retention_days" owns {"days"}.
type SetSettingInput struct {
	Enabled *bool `json:"enabled,omitempty"`
	Days    *int  `json:"days,omitempty"`
}

// SetSetting upserts one runtime setting (key: vip | vip_retention_days).
func (c *Client) SetSetting(ctx context.Context, key string, in SetSettingInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPut, adminPath("settings", url.PathEscape(key)), nil, in, true)
}
