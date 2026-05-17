// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// CreatePostResponse represents a post creation response.
type CreatePostResponse struct {
	ID string `json:"id"`
}

// PostListItem represents a post in list responses.
type PostListItem struct {
	ID        int       `json:"id"`
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// PostsListResponse represents a list of posts response.
type PostsListResponse struct {
	Posts      []PostListItem `json:"posts"`
	Pagination Pagination     `json:"pagination"`
}

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
	GetUserPosts(ctx context.Context, userID int, offset, limit int) ([]post.Post, int64, error)
}

// CreatePost returns a handler for creating a new post.
func CreatePost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
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

		c.JSON(http.StatusCreated, CreatePostResponse{ID: id})
	}
}

func respondPostError(c *gin.Context, err error) {
	if se, ok := service.AsServiceError(err); ok && se.Code == service.ErrNotFound {
		c.String(http.StatusNotFound, getI18nMessage(c, "Not Found"))
		return
	}
	log.Printf("RenderPost error: %v", err)
	c.String(http.StatusInternalServerError, getI18nMessage(c, "Failed to render post"))
}

// RenderPost returns a handler for rendering a post.
func RenderPost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if c.Query("format") == "raw" {
			title, body, err := postSvc.GetPostMarkdown(c.Request.Context(), id)
			if err != nil {
				respondPostError(c, err)
				return
			}

			content := "# " + title + "\n\n" + body
			c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(content))
			return
		}

		title, htmlContent, err := postSvc.RenderPostHTML(c.Request.Context(), id)
		if err != nil {
			respondPostError(c, err)
			return
		}
		c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
	}
}

func newPostListItem(p post.Post) PostListItem {
	return PostListItem{
		ID:        p.ID,
		QID:       p.QID,
		Title:     p.Title,
		CreatedAt: p.CreatedAt,
	}
}

// PostsList returns a handler for listing user's posts.
func PostsList(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
			return
		}

		query, valid := bindPaginationQuery(c)
		if !valid {
			return
		}

		posts, total, err := postSvc.GetUserPosts(c.Request.Context(), u.ID, query.Offset, query.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := mapSlice(posts, newPostListItem)

		c.JSON(http.StatusOK, PostsListResponse{
			Posts:      items,
			Pagination: query.ToPagination(total),
		})
	}
}
