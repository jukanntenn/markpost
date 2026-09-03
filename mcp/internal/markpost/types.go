package markpost

import (
	"net/url"
	"strconv"
)

// Typed DTOs are declared only where the client itself reads fields (tokens,
// post key, created id). Everything else passes through as raw JSON.

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// postKeyResponse mirrors GET /api/v1/post-key.
type postKeyResponse struct {
	PostKey string `json:"post_key"`
}

// createPostRequest mirrors POST /{post_key}: title and markdown body.
type createPostRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// createPostResponse mirrors the 201 body of POST /{post_key}.
type createPostResponse struct {
	ID string `json:"id"`
}

// changePasswordRequest mirrors POST /api/v1/auth/change-password.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// CreatePostResult is the client-composed create_post payload: the backend's
// {id} plus the public render URL derived from the instance base URL.
type CreatePostResult struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// paginationQuery builds the shared page/limit query (limit ≤ 100, the
// backend's hard cap); zero values are omitted so the backend applies its
// defaults (page 1, limit 20).
func paginationQuery(page, limit int) url.Values {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return q
}
