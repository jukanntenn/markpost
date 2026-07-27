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

// UserResponse represents the user data returned in API responses.
type UserResponse struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
	Role      string  `json:"role"`
}

func newUserResponse(u user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Role:      string(u.Role),
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
// skips verification when no password is set.
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
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
	ID          int64     `json:"id"`
	Status      string    `json:"status"`
	LastError   string    `json:"last_error"`
	CreatedAt   time.Time `json:"created_at"`
	ChannelID   *int      `json:"channel_id"`
	PostTitle   *string   `json:"post_title"`
	PostQID     *string   `json:"post_qid"`
	ChannelName *string   `json:"channel_name"`
	Username    *string   `json:"username"`
}

func newDeliveryHistoryItem(h *delivery.HistoryRow) DeliveryHistoryItem {
	return DeliveryHistoryItem{
		ID:          h.ID,
		Status:      deliveryStatusName(h.Status),
		LastError:   h.LastError,
		CreatedAt:   h.CreatedAt,
		ChannelID:   h.ChannelID,
		PostTitle:   h.PostTitle,
		PostQID:     h.PostQID,
		ChannelName: h.ChannelName,
		Username:    h.Username,
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

// AdminUserItem represents a user entry in the admin user list.
type AdminUserItem struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func newAdminUserItem(u user.User) AdminUserItem {
	return AdminUserItem{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      string(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
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
	Search string `form:"search"`
}

// DeliveryHistoryQuery binds the query parameters for a user's delivery history
// listing: pagination plus an optional channel_id filter (0 or absent = no
// channel filter).
type DeliveryHistoryQuery struct {
	PaginationQuery
	ChannelID int `form:"channel_id"`
}

// PaginatedUsers represents a paginated list of admin user items.
type PaginatedUsers struct {
	Users      []AdminUserItem `json:"users"`
	Pagination Pagination      `json:"pagination"`
}

// PaginatedPosts represents a paginated list of admin post items.
type PaginatedPosts struct {
	Posts      []AdminPostItem `json:"posts"`
	Pagination Pagination      `json:"pagination"`
}

// PaginatedChannels represents a paginated list of admin channel items.
type PaginatedChannels struct {
	Channels   []AdminChannelItem `json:"channels"`
	Pagination Pagination         `json:"pagination"`
}

// --- Health types ---

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// --- Audit types ---

// AdminAuditLogItem represents an audit log entry in the admin audit log list.
type AdminAuditLogItem struct {
	ID         int64           `json:"id"`
	ActorID    int             `json:"actor_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Metadata   json.RawMessage `json:"metadata"`
	IP         string          `json:"ip"`
	CreatedAt  time.Time       `json:"created_at"`
}

func newAdminAuditLogItem(log audit.Log) AdminAuditLogItem {
	return AdminAuditLogItem{
		ID:         log.ID,
		ActorID:    log.ActorID,
		Action:     log.Action,
		TargetType: log.TargetType,
		TargetID:   log.TargetID,
		Metadata:   log.Metadata,
		IP:         log.IP,
		CreatedAt:  log.CreatedAt,
	}
}

// PaginatedAuditLogs represents a paginated list of admin audit log items.
type PaginatedAuditLogs struct {
	AuditLogs  []AdminAuditLogItem `json:"audit_logs"`
	Pagination Pagination          `json:"pagination"`
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
