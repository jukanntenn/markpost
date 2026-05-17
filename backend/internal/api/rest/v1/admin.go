// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// AdminService provides admin operations.
type AdminService interface {
	ListAllUsers(ctx context.Context, offset, limit int) ([]user.User, int64, error)
	ListAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error)
	ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
}

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
	ID        string    `json:"id"`
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func newAdminPostItem(p post.Post) AdminPostItem {
	username := ""
	if p.User.ID != 0 {
		username = p.User.Username
	}
	return AdminPostItem{
		ID:        p.QID,
		QID:       p.QID,
		Title:     p.Title,
		UserID:    p.UserID,
		Username:  username,
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

type AdminUsersResponse struct {
	Users      []AdminUserItem `json:"users"`
	Pagination Pagination      `json:"pagination"`
}

type AdminPostsResponse struct {
	Posts      []AdminPostItem `json:"posts"`
	Pagination Pagination      `json:"pagination"`
}

type AdminChannelsResponse struct {
	Channels   []AdminChannelItem `json:"channels"`
	Pagination Pagination         `json:"pagination"`
}

type AdminPostsQuery struct {
	PaginationQuery
	Search string `form:"search"`
}

// AdminListUsers returns a handler for listing all users.
func AdminListUsers(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		query, ok := bindPaginationQuery(c)
		if !ok {
			return
		}

		users, total, err := adminSvc.ListAllUsers(c.Request.Context(), query.Offset, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := mapSlice(users, newAdminUserItem)

		c.JSON(http.StatusOK, AdminUsersResponse{
			Users:      items,
			Pagination: query.ToPagination(total),
		})
	}
}

// AdminListPosts returns a handler for listing all posts.
func AdminListPosts(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var query AdminPostsQuery
		if !bindQuery(c, &query) || !validatePaginationQuery(c, &query.PaginationQuery) {
			return
		}

		posts, total, err := adminSvc.ListAllPosts(c.Request.Context(), query.Search, query.Offset, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := mapSlice(posts, newAdminPostItem)

		c.JSON(http.StatusOK, AdminPostsResponse{
			Posts:      items,
			Pagination: query.PaginationQuery.ToPagination(total),
		})
	}
}

// AdminListChannels returns a handler for listing all delivery channels.
func AdminListChannels(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		query, ok := bindPaginationQuery(c)
		if !ok {
			return
		}

		channels, total, err := adminSvc.ListAllDeliveryChannels(c.Request.Context(), query.Offset, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := mapSlice(channels, newAdminChannelItem)

		c.JSON(http.StatusOK, AdminChannelsResponse{
			Channels:   items,
			Pagination: query.ToPagination(total),
		})
	}
}
