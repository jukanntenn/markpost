// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// AdminService provides admin operations.
type AdminService interface {
	ListAllUsers(ctx context.Context, page, limit int) ([]user.User, int64, error)
	ListAllPosts(ctx context.Context, search string, page, limit int) ([]post.Post, int64, error)
	ListAllDeliveryChannels(ctx context.Context, page, limit int) ([]delivery.Channel, int64, error)
}

// AdminListUsers returns a handler for listing all users.
func AdminListUsers(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		type queryParams struct {
			Page  int `form:"page" binding:"omitempty,min=1"`
			Limit int `form:"limit" binding:"omitempty,min=1"`
		}
		var query queryParams
		if !bindQuery(c, &query) {
			return
		}

		query.Page = defaultInt(query.Page, 1)
		query.Limit = defaultInt(query.Limit, 20)
		if query.Limit > 100 {
			apierr.RespondError(c, service.NewServiceError(service.ErrInvalidRequest, "invalid limit"))
			return
		}

		users, total, err := adminSvc.ListAllUsers(c.Request.Context(), query.Page, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := make([]gin.H, 0, len(users))
		for _, u := range users {
			items = append(items, gin.H{
				"id":         u.ID,
				"username":   u.Username,
				"email":      u.Email,
				"role":       u.Role,
				"is_active":  u.IsActive,
				"created_at": u.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"users": items, "total": total})
	}
}

// AdminListPosts returns a handler for listing all posts.
func AdminListPosts(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		type queryParams struct {
			Search string `form:"search"`
			Page   int    `form:"page" binding:"omitempty,min=1"`
			Limit  int    `form:"limit" binding:"omitempty,min=1"`
		}
		var query queryParams
		if !bindQuery(c, &query) {
			return
		}

		query.Page = defaultInt(query.Page, 1)
		query.Limit = defaultInt(query.Limit, 20)
		if query.Limit > 100 {
			apierr.RespondError(c, service.NewServiceError(service.ErrInvalidRequest, "invalid limit"))
			return
		}

		posts, total, err := adminSvc.ListAllPosts(c.Request.Context(), query.Search, query.Page, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := make([]gin.H, 0, len(posts))
		for _, p := range posts {
			username := ""
			if p.User.ID != 0 {
				username = p.User.Username
			}
			items = append(items, gin.H{
				"id":         p.QID,
				"qid":        p.QID,
				"title":      p.Title,
				"user_id":    p.UserID,
				"username":   username,
				"created_at": p.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"posts": items, "total": total})
	}
}

// AdminListChannels returns a handler for listing all delivery channels.
func AdminListChannels(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		type queryParams struct {
			Page  int `form:"page" binding:"omitempty,min=1"`
			Limit int `form:"limit" binding:"omitempty,min=1"`
		}
		var query queryParams
		if !bindQuery(c, &query) {
			return
		}

		query.Page = defaultInt(query.Page, 1)
		query.Limit = defaultInt(query.Limit, 20)
		if query.Limit > 100 {
			apierr.RespondError(c, service.NewServiceError(service.ErrInvalidRequest, "invalid limit"))
			return
		}

		channels, total, err := adminSvc.ListAllDeliveryChannels(c.Request.Context(), query.Page, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := make([]gin.H, 0, len(channels))
		for _, ch := range channels {
			items = append(items, gin.H{
				"id":         ch.ID,
				"name":       ch.Name,
				"type":       string(ch.Kind),
				"enabled":    ch.Enabled,
				"user_id":    ch.UserID,
				"webhook_url": ch.WebhookURL,
				"created_at": ch.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"channels": items, "total": total})
	}
}
