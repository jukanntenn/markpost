// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"markpost/internal/domain/post"
	"markpost/internal/service"
	postsvc "markpost/internal/service/post"
	"markpost/pkg/apierr"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// PostRequest represents a post creation request.
type PostRequest struct {
	Title string `json:"title" binding:"required,titlesize"`
	Body  string `json:"body" binding:"required,bodysize"`
}

// PostService provides post operations.
type PostService interface {
	CreatePost(ctx context.Context, title, body string, userID int) (string, error)
	RenderPostHTML(ctx context.Context, qid string) (string, string, error)
	GetPostMarkdown(ctx context.Context, qid string) (string, string, error)
	GetUserPosts(ctx context.Context, userID int, page, limit int) ([]post.Post, int64, error)
}

// CreatePost returns a handler for creating a new post.
func CreatePost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

		var req PostRequest
		if !bindJSON(c, &req) {
			return
		}

		id, err := postSvc.CreatePost(c.Request.Context(), req.Title, req.Body, u.ID)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

// getI18nMessage gets an i18n message or returns a default value if i18n is not available.
func getI18nMessage(c *gin.Context, defaultMsg string) string {
	if _, exists := c.Get("i18n"); exists {
		return ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    defaultMsg,
				Other: defaultMsg,
			},
		})
	}
	return defaultMsg
}

// RenderPost returns a handler for rendering a post.
func RenderPost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if c.Query("format") == "raw" {
			title, body, err := postSvc.GetPostMarkdown(c.Request.Context(), id)
			if err != nil {
				if se, ok := postsvc.AsServiceError(err); ok && se.Code == postsvc.ErrNotFound {
					c.String(http.StatusNotFound, getI18nMessage(c, "Not Found"))
				} else {
					c.String(http.StatusInternalServerError, getI18nMessage(c, "Failed to render post"))
				}
				return
			}

			content := "# " + title + "\n\n" + body
			c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(content))
			return
		}

		title, htmlContent, err := postSvc.RenderPostHTML(c.Request.Context(), id)
		if err != nil {
			if se, ok := postsvc.AsServiceError(err); ok && se.Code == postsvc.ErrNotFound {
				c.String(http.StatusNotFound, getI18nMessage(c, "Not Found"))
			} else {
				log.Printf("RenderPost error: %v", err)
				c.String(http.StatusInternalServerError, getI18nMessage(c, "Failed to render post"))
			}
			return
		}
		c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
	}
}

// PostsList returns a handler for listing user's posts.
func PostsList(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

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
			err := service.NewServiceErrorWrap(service.ErrInvalidRequest, "invalid limit", nil)
			apierr.RespondError(c, err)
			return
		}

		posts, total, err := postSvc.GetUserPosts(c.Request.Context(), u.ID, query.Page, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := make([]gin.H, 0, len(posts))
		for _, p := range posts {
			items = append(items, gin.H{
				"id":         p.ID,
				"qid":        p.QID,
				"title":      p.Title,
				"created_at": p.CreatedAt,
			})
		}
		totalPages := (total + int64(query.Limit) - 1) / int64(query.Limit)

		c.JSON(http.StatusOK, gin.H{
			"posts": items,
			"pagination": gin.H{
				"page":        query.Page,
				"limit":       query.Limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
	}
}
