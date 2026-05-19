package v1

import (
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	delivery_svc "markpost/internal/service/delivery"
	"markpost/pkg/utils"
)

// --- Auth types ---

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

type TokenFields struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthResponse struct {
	User UserResponse `json:"user"`
	TokenFields
}

type RefreshTokenResponse struct {
	TokenFields
}

type PostKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

type GitHubLoginRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

type UsernameLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// --- Post types ---

type CreatePostResponse struct {
	ID string `json:"id"`
}

type PostListItem struct {
	ID        int       `json:"id"`
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type PostsListResponse struct {
	Posts      []PostListItem `json:"posts"`
	Pagination Pagination     `json:"pagination"`
}

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

type ChannelResponse struct {
	ID         int                  `json:"id"`
	Kind       delivery.ChannelKind `json:"kind"`
	Name       string               `json:"name"`
	Enabled    bool                 `json:"enabled"`
	WebhookURL string               `json:"webhook_url"`
	Keywords   string               `json:"keywords"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

func newChannelResponse(ch delivery.Channel) ChannelResponse {
	return ChannelResponse{
		ID:         ch.ID,
		Kind:       ch.Kind,
		Name:       ch.Name,
		Enabled:    ch.Enabled,
		WebhookURL: ch.WebhookURL,
		Keywords:   ch.Keywords,
		CreatedAt:  ch.CreatedAt,
		UpdatedAt:  ch.UpdatedAt,
	}
}

type ChannelsListResponse struct {
	Channels []ChannelResponse `json:"channels"`
}

type SingleChannelResponse struct {
	Channel ChannelResponse `json:"channel"`
}

type CreateDeliveryChannelRequest struct {
	Kind       string `json:"kind" binding:"required"`
	Name       string `json:"name" binding:"required"`
	WebhookURL string `json:"webhook_url" binding:"required"`
	Keywords   string `json:"keywords"`
}

func (r CreateDeliveryChannelRequest) toParams() delivery_svc.UpdateChannelParams {
	return delivery_svc.UpdateChannelParams{
		Kind:       r.Kind,
		Name:       r.Name,
		WebhookURL: r.WebhookURL,
		Keywords:   r.Keywords,
	}
}

type UpdateDeliveryChannelRequest struct {
	Kind       *string `json:"kind"`
	Name       *string `json:"name"`
	WebhookURL *string `json:"webhook_url"`
	Keywords   *string `json:"keywords"`
	Enabled    *bool   `json:"enabled"`
}

func (r UpdateDeliveryChannelRequest) toParams() delivery_svc.UpdateChannelParams {
	return delivery_svc.UpdateChannelParams{
		Kind:       utils.Deref(r.Kind),
		Name:       utils.Deref(r.Name),
		WebhookURL: utils.Deref(r.WebhookURL),
		Keywords:   utils.Deref(r.Keywords),
		Enabled:    r.Enabled,
	}
}

// --- Admin types ---

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

type AdminChannelItem struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	Enabled    bool      `json:"enabled"`
	UserID     int       `json:"user_id"`
	WebhookURL string    `json:"webhook_url"`
	CreatedAt  time.Time `json:"created_at"`
}

func newAdminChannelItem(ch delivery.Channel) AdminChannelItem {
	return AdminChannelItem{
		ID:         ch.ID,
		Name:       ch.Name,
		Kind:       string(ch.Kind),
		Enabled:    ch.Enabled,
		UserID:     ch.UserID,
		WebhookURL: ch.WebhookURL,
		CreatedAt:  ch.CreatedAt,
	}
}

type AdminPostsQuery struct {
	PaginationQuery
	Search string `form:"search"`
}

// --- Health types ---

type HealthResponse struct {
	Status string `json:"status"`
}
