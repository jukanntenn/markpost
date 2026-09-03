package api

import "time"

// Wire types mirroring the server's REST v1 DTOs
// (backend/internal/api/rest/v1/types.go). Field names and shapes must track
// the server contract; this package re-declares them instead of importing the
// server module so the CLI builds and ships as an independent tool.

type User struct {
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

// TokenPair is the token block embedded in login and refresh responses.
type TokenPair struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type LoginResponse struct {
	User User `json:"user"`
	TokenPair
}

type RefreshResponse struct {
	TokenPair
}

type PostKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

type RotatePostKeyResponse struct {
	PostKey string `json:"post_key"`
}

// PostList is the flat paginated envelope every list endpoint returns:
// {items, total, page, limit, total_pages}.
type PostList struct {
	Items      []PostListItem `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}

type PostListItem struct {
	ID        int       `json:"id"`
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePostResponse struct {
	ID string `json:"id"`
}

// Retention is the caller's effective retention policy; zero days means
// keep forever.
type Retention struct {
	PostsDays   int `json:"posts_days"`
	HistoryDays int `json:"history_days"`
}

type VersionResponse struct {
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
