package v1

import (
	"encoding/json"
	"time"

	"markpost/internal/domain/audit"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	delivery_svc "markpost/internal/service/delivery"
	"markpost/pkg/utils"
)

// --- Auth types ---

// UserResponse represents the user data returned in API responses. is_active /
// is_email_verified let the frontend sense ban state (B1.12).
type UserResponse struct {
	ID              int     `json:"id"`
	Email           string  `json:"email"`
	Username        string  `json:"username"`
	Name            string  `json:"name"`
	AvatarURL       *string `json:"avatar_url"`
	Role            string  `json:"role"`
	IsActive        bool    `json:"is_active"`
	IsEmailVerified bool    `json:"is_email_verified"`
	VIP             bool    `json:"vip"`
}

func newUserResponse(u user.User) UserResponse {
	return UserResponse{
		ID:              u.ID,
		Email:           u.Email,
		Username:        u.Username,
		Name:            u.Name,
		AvatarURL:       u.AvatarURL,
		Role:            string(u.Role),
		IsActive:        u.IsActive,
		IsEmailVerified: u.IsEmailVerified,
		VIP:             u.VIP,
	}
}

// TokenFields represents JWT token fields returned in authentication responses.
type TokenFields struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// AuthResponse represents the response for a successful authentication.
type AuthResponse struct {
	User UserResponse `json:"user"`
	TokenFields
}

// RefreshTokenResponse represents the response for a successful token refresh.
type RefreshTokenResponse struct {
	TokenFields
}

// ChangePasswordResponse is the change-password success body: a fresh token
// pair so the client continues seamlessly (C2.2, no re-login).
type ChangePasswordResponse struct {
	TokenFields
}

// SessionsResponse is the user/admin session listing body (I.12/D3.2).
type SessionsResponse struct {
	Sessions []user.RefreshToken `json:"sessions"`
}

// PostKeyResponse represents the response containing a user's post key.
type PostKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

// OAuthURLResponse represents the response containing a GitHub OAuth authorization URL.
type OAuthURLResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

// GitHubLoginRequest represents the request body for GitHub OAuth login.
type GitHubLoginRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// UsernameLoginRequest represents the request body for username and password login.
type UsernameLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents the request body for refreshing an authentication token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// PasswordChangeRequest represents the request body for changing a user's password.
// CurrentPassword is optional: users created via OAuth without a local password
// may leave it empty; the service layer validates it against the stored hash and
// skips verification when no password is set. Length policy (min 8/max 72) is
// enforced by the service layer — C2.3 单一真相源, no binding tag here.
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// --- Post types ---

// CreatePostResponse represents the response for a successful post creation.
type CreatePostResponse struct {
	ID string `json:"id"`
}

// PostListItem represents a single post entry in a paginated post list.
type PostListItem struct {
	ID        int       `json:"id"`
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// PostsListResponse is the flat paginated list returned by GET /api/v1/posts.
type PostsListResponse struct {
	Items      []PostListItem `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}

// PostsQuery binds the user posts listing query (B3.3 search + pagination).
type PostsQuery struct {
	PaginationQuery
	Search string `form:"search"`
}

// PostRequest represents the request body for creating a new post.
type PostRequest struct {
	Title string `json:"title" binding:"required,titlesize"`
	Body  string `json:"body" binding:"required,bodysize"`
}

func newPostListItem(p post.Post) PostListItem {
	return PostListItem{
		ID:        p.ID,
		QID:       p.QID,
		Title:     p.Title,
		CreatedAt: p.CreatedAt,
	}
}

// --- Delivery types ---

// ChannelResponse represents a delivery channel in API responses.
type ChannelResponse struct {
	ID            int                           `json:"id"`
	Kind          delivery.ChannelKind          `json:"kind"`
	Name          string                        `json:"name"`
	Enabled       bool                          `json:"enabled"`
	Configuration delivery.ChannelConfiguration `json:"configuration"`
	Keywords      string                        `json:"keywords"`
	CreatedAt     time.Time                     `json:"created_at"`
	UpdatedAt     time.Time                     `json:"updated_at"`
}

func newChannelResponse(ch delivery.Channel) ChannelResponse {
	return ChannelResponse{
		ID:            ch.ID,
		Kind:          ch.Kind,
		Name:          ch.Name,
		Enabled:       ch.Enabled,
		Configuration: ch.Configuration,
		Keywords:      ch.Keywords,
		CreatedAt:     ch.CreatedAt,
		UpdatedAt:     ch.UpdatedAt,
	}
}

// ChannelsListResponse represents a list of delivery channels.
type ChannelsListResponse struct {
	Items []ChannelResponse `json:"items"`
}

// SingleChannelResponse represents a response containing a single delivery channel.
type SingleChannelResponse struct {
	Channel ChannelResponse `json:"channel"`
}

// CreateDeliveryChannelRequest represents the request body for creating a delivery channel.
type CreateDeliveryChannelRequest struct {
	Kind          string          `json:"kind" binding:"required"`
	Name          string          `json:"name" binding:"required"`
	Configuration json.RawMessage `json:"configuration" binding:"required"`
	Keywords      string          `json:"keywords"`
}

func (r CreateDeliveryChannelRequest) toParams() delivery_svc.UpdateChannelParams {
	return delivery_svc.UpdateChannelParams{
		Kind:          r.Kind,
		Name:          r.Name,
		Configuration: r.Configuration,
		Keywords:      &r.Keywords,
	}
}

// UpdateDeliveryChannelRequest represents the request body for updating a delivery channel.
type UpdateDeliveryChannelRequest struct {
	Kind          *string          `json:"kind"`
	Name          *string          `json:"name"`
	Configuration *json.RawMessage `json:"configuration"`
	Keywords      *string          `json:"keywords"`
	Enabled       *bool            `json:"enabled"`
}

func (r UpdateDeliveryChannelRequest) toParams() delivery_svc.UpdateChannelParams {
	params := delivery_svc.UpdateChannelParams{
		Kind:     utils.Deref(r.Kind),
		Name:     utils.Deref(r.Name),
		Keywords: r.Keywords,
		Enabled:  r.Enabled,
	}
	if r.Configuration != nil {
		params.Configuration = *r.Configuration
	}
	return params
}

// DeliveryHistoryItem represents a delivery history entry in API responses. The
// nullable pointers reflect ON DELETE SET NULL: a nil field means the referenced
// post/channel/user was deleted.
type DeliveryHistoryItem struct {
	ID            int64     `json:"id"`
	Status        string    `json:"status"`
	LastError     string    `json:"last_error"`
	ErrorCategory string    `json:"error_category"`
	CreatedAt     time.Time `json:"created_at"`
	ChannelID     *int      `json:"channel_id"`
	PostTitle     *string   `json:"post_title"`
	PostQID       *string   `json:"post_qid"`
	ChannelName   *string   `json:"channel_name"`
	Username      *string   `json:"username"`
}

func newDeliveryHistoryItem(h *delivery.HistoryRow) DeliveryHistoryItem {
	return DeliveryHistoryItem{
		ID:            h.ID,
		Status:        deliveryStatusName(h.Status),
		LastError:     h.LastError,
		ErrorCategory: h.ErrorCategory,
		CreatedAt:     h.CreatedAt,
		ChannelID:     h.ChannelID,
		PostTitle:     h.PostTitle,
		PostQID:       h.PostQID,
		ChannelName:   h.ChannelName,
		Username:      h.Username,
	}
}

func deliveryStatusName(s delivery.Status) string {
	switch s {
	case delivery.StatusDelivered:
		return "delivered"
	case delivery.StatusFailed:
		return "failed"
	case delivery.StatusExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// DeliveryHistoryListResponse represents a paginated list of delivery history.
type DeliveryHistoryListResponse struct {
	History    []DeliveryHistoryItem `json:"history"`
	Pagination Pagination            `json:"pagination"`
}

// DeliveryLatestListResponse represents the most recent delivery per channel
// (one item per channel that has any history).
type DeliveryLatestListResponse struct {
	Items []DeliveryHistoryItem `json:"items"`
}

// --- Admin types ---

// AdminUserItem represents a user entry in the admin user list / detail
// (D3.1/D3.2): post_key + last_login_at exposed for the detail profile page.
type AdminUserItem struct {
	ID              int        `json:"id"`
	Username        string     `json:"username"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	IsEmailVerified bool       `json:"is_email_verified"`
	GitHubID        *int64     `json:"github_id"`
	Role            string     `json:"role"`
	IsActive        bool       `json:"is_active"`
	VIP             bool       `json:"vip"`
	RetentionDays   *int       `json:"retention_days"`
	PostKey         string     `json:"post_key"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func newAdminUserItem(u user.User) AdminUserItem {
	return AdminUserItem{
		ID:              u.ID,
		Username:        u.Username,
		Name:            u.Name,
		Email:           u.Email,
		IsEmailVerified: u.IsEmailVerified,
		GitHubID:        u.GitHubID,
		Role:            string(u.Role),
		IsActive:        u.IsActive,
		VIP:             u.VIP,
		RetentionDays:   u.RetentionDays,
		PostKey:         u.PostKey,
		LastLoginAt:     u.LastLoginAt,
		CreatedAt:       u.CreatedAt,
	}
}

// AdminPostItem represents a post entry in the admin post list.
type AdminPostItem struct {
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func newAdminPostItem(p post.Post) AdminPostItem {
	return AdminPostItem{
		QID:       p.QID,
		Title:     p.Title,
		UserID:    p.UserID,
		Username:  p.User.Username,
		CreatedAt: p.CreatedAt,
	}
}

// AdminChannelItem represents a delivery channel entry in the admin channel list.
type AdminChannelItem struct {
	ID            int                           `json:"id"`
	Name          string                        `json:"name"`
	Kind          string                        `json:"kind"`
	Enabled       bool                          `json:"enabled"`
	UserID        int                           `json:"user_id"`
	Username      string                        `json:"username"`
	Configuration delivery.ChannelConfiguration `json:"configuration"`
	CreatedAt     time.Time                     `json:"created_at"`
}

func newAdminChannelItem(ch delivery.Channel) AdminChannelItem {
	username := ""
	if ch.User.ID > 0 {
		username = ch.User.Username
	}
	return AdminChannelItem{
		ID:            ch.ID,
		Name:          ch.Name,
		Kind:          string(ch.Kind),
		Enabled:       ch.Enabled,
		UserID:        ch.UserID,
		Username:      username,
		Configuration: ch.Configuration,
		CreatedAt:     ch.CreatedAt,
	}
}

// AdminPostsQuery represents the query parameters for admin post listing.
type AdminPostsQuery struct {
	PaginationQuery
	Search   string `form:"search"`
	Username string `form:"username"`
}

// AdminUsersQuery represents the query parameters for the admin user listing
// (D3.1 username LIKE search).
type AdminUsersQuery struct {
	PaginationQuery
	Search string `form:"search"`
}

// AdminAuditQuery binds the admin audit log filters (D4.3).
type AdminAuditQuery struct {
	PaginationQuery
	ActorID    int    `form:"actor_id"`
	Action     string `form:"action"`
	TargetType string `form:"target_type"`
	TargetID   string `form:"target_id"`
	Since      string `form:"since"` // RFC3339
	Until      string `form:"until"` // RFC3339
}

// DeliveryHistoryQuery binds the query parameters for a user's delivery history
// listing: pagination plus optional channel_id / status filters (B3.4).
type DeliveryHistoryQuery struct {
	PaginationQuery
	ChannelID int    `form:"channel_id"`
	Status    string `form:"status"`
}

// AdminDeliveryHistoryQuery binds the query parameters for the admin delivery
// history listing: user_id / channel_id / status / error_category filters.
type AdminDeliveryHistoryQuery struct {
	PaginationQuery
	UserID        int    `form:"user_id"`
	ChannelID     int    `form:"channel_id"`
	Status        string `form:"status"`
	ErrorCategory string `form:"error_category"`
}

// DeliveryStatsResponse is the user delivery stats body (B2.7/K.2):
// today counters for the pipeline status bar plus the per-day trend.
type DeliveryStatsResponse struct {
	Today delivery.TodayCounts  `json:"today"`
	Trend []*delivery.DailyStat `json:"trend"`
}

// PendingAttemptsResponse lists the user's in-flight attempts (K.2).
type PendingAttemptsResponse struct {
	Items []*delivery.PendingAttemptRow `json:"items"`
}

// AdminLockedChannelsResponse lists channels flagged by the failing-channel
// query (D2.1/K.7).
type AdminLockedChannelsResponse struct {
	Items []*delivery.LockedChannel `json:"items"`
}

// RotatePostKeyResponse returns the new post key (C2.5).
type RotatePostKeyResponse struct {
	PostKey string `json:"post_key"`
}

// PaginatedItemsResponse is the flat paginated envelope mandated by I.10:
// { items, total, page, limit, total_pages }. The items field carries the
// resource-specific item type; swag documents it as an opaque object.
type PaginatedItemsResponse struct {
	Items      any   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// --- Health types ---

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// VersionResponse reports the build version of the running binary. The value
// is the VERSION build-arg (git describe / git tag), injected at route
// registration; post-deploy verification compares it against the expected
// version to catch a container that is up but still running the old image.
type VersionResponse struct {
	Version string `json:"version"`
}

// --- Audit types ---

// AdminAuditLogItem represents an audit log entry in the admin audit log list.
// actor_username is JOINed at read time (D4.1). target_username is JOINed only
// for user-targeted rows (DEV-1); nil otherwise.
type AdminAuditLogItem struct {
	ID             int64           `json:"id"`
	ActorID        int             `json:"actor_id"`
	ActorUsername  string          `json:"actor_username"`
	Action         string          `json:"action"`
	TargetType     string          `json:"target_type"`
	TargetID       string          `json:"target_id"`
	TargetUsername *string         `json:"target_username"`
	Metadata       json.RawMessage `json:"metadata"`
	IP             string          `json:"ip"`
	CreatedAt      time.Time       `json:"created_at"`
}

func newAdminAuditLogItem(row audit.LogRow) AdminAuditLogItem {
	return AdminAuditLogItem{
		ID:             row.ID,
		ActorID:        row.ActorID,
		ActorUsername:  row.ActorUsername,
		Action:         row.Action,
		TargetType:     row.TargetType,
		TargetID:       row.TargetID,
		TargetUsername: row.TargetUsername,
		Metadata:       row.Metadata,
		IP:             row.IP,
		CreatedAt:      row.CreatedAt,
	}
}

// AdminSessionItem represents a user session in the admin session list.
type AdminSessionItem struct {
	ID        int64     `json:"id"`
	TokenHash string    `json:"token_hash"`
	Revoked   bool      `json:"revoked"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminSessionsResponse represents a list of user sessions.
type AdminSessionsResponse struct {
	Sessions []AdminSessionItem `json:"sessions"`
}

// AdminStatsResponse represents the aggregate counts shown on the admin
// dashboard, including the 7-day deltas (D2.4).
type AdminStatsResponse struct {
	Counts AdminStatsCounts `json:"counts"`
}

// AdminStatsCounts holds the per-resource totals and week deltas.
type AdminStatsCounts struct {
	Users            int64 `json:"users"`
	Posts            int64 `json:"posts"`
	Channels         int64 `json:"channels"`
	History          int64 `json:"history"`
	UsersWeekDelta   int64 `json:"users_week_delta"`
	PostsWeekDelta   int64 `json:"posts_week_delta"`
	HistoryWeekDelta int64 `json:"history_week_delta"`
}

// AdminResetPasswordResponse returns the generated temporary password in
// plaintext exactly once (D3.3 方案 B).
type AdminResetPasswordResponse struct {
	Password string `json:"password"`
}
