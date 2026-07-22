package v1

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"markpost/internal/apierr"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/web"

	"github.com/gin-gonic/gin"
)

// PostService defines the interface for post-related operations.
type PostService interface {
	CreatePost(ctx context.Context, title, body string, userID int) (string, error)
	RenderPostHTML(ctx context.Context, qid string) (title, html, etag string, createdAt time.Time, err error)
	GetPostMarkdown(ctx context.Context, qid string) (title, body, etag string, createdAt time.Time, err error)
	GetUserPosts(ctx context.Context, userID int, offset, limit int) ([]post.Post, int64, error)
	DeletePostByQID(ctx context.Context, qid string, ownerID int) error
}

// CreatePost godoc
// @Summary Create a new post with a post key
// @Tags posts
// @Accept json
// @Produce json
// @Param post_key path string true "Post key used for authentication"
// @Param body body PostRequest true "Post title and markdown body"
// @Success 201 {object} CreatePostResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /{post_key} [post]
func CreatePost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
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
		})
	}
}

// RenderPost godoc
// @Summary Render a post as HTML or raw markdown
// @Tags posts
// @Produce html
// @Param id path string true "Post QID"
// @Param format query string false "Response format (raw returns markdown)"
// @Success 200 {string} string ""
// @Failure 404 {object} apierr.ErrorResponse
// @Router /{id} [get]
func RenderPost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		isRaw := c.Query("format") == "raw"

		setCacheHeaders := func(etag string, createdAt time.Time) {
			c.Header("ETag", `"`+etag+`"`)
			c.Header("Cache-Control", "public, max-age=300, s-maxage=3600")
			c.Header("Cache-Tag", "post-"+id)
			c.Header("Vary", "Accept-Encoding")
			if !createdAt.IsZero() {
				c.Header("Last-Modified", createdAt.UTC().Format(http.TimeFormat))
			}
		}

		if isRaw {
			title, body, etag, createdAt, err := postSvc.GetPostMarkdown(c.Request.Context(), id)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			setCacheHeaders(etag, createdAt)
			if etagMatch(c.GetHeader("If-None-Match"), etag) {
				c.AbortWithStatus(http.StatusNotModified)
				return
			}
			c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte("# "+title+"\n\n"+body))
			return
		}

		title, htmlContent, etag, createdAt, err := postSvc.RenderPostHTML(c.Request.Context(), id)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		setCacheHeaders(etag, createdAt)
		if etagMatch(c.GetHeader("If-None-Match"), etag) {
			c.AbortWithStatus(http.StatusNotModified)
			return
		}
		c.HTML(http.StatusOK, "post.html", gin.H{
			"Title":   title,
			"Body":    template.HTML(htmlContent),
			"CSSHash": web.CSSHash,
		})
	}
}

// PostsList godoc
// @Summary List the current user's posts
// @Tags posts
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} PostsListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/posts [get]
func PostsList(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUserPaginatedQuery(c, postSvc.GetUserPosts, newPostListItem,
			paginatedWrap[PostListItem]("posts"),
		)
	}
}

// DeleteOwnPost godoc
// @Summary Delete a post owned by the current user
// @Tags posts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Post QID"
// @Success 204 {string} string ""
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/posts/{id} [delete]
func DeleteOwnPost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			qid := c.Param("id")
			if err := postSvc.DeletePostByQID(c.Request.Context(), qid, u.ID); err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.Status(http.StatusNoContent)
		})
	}
}

// DeleteAnyPost godoc
// @Summary Delete any post (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path string true "Post QID"
// @Success 204 {string} string ""
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/posts/{id} [delete]
func DeleteAnyPost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		qid := c.Param("id")
		if err := postSvc.DeletePostByQID(c.Request.Context(), qid, 0); err != nil {
			apierr.RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
