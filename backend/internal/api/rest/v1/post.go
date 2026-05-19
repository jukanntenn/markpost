package v1

import (
	"context"
	"html/template"
	"net/http"

	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

type PostService interface {
	CreatePost(ctx context.Context, title, body string, userID int) (string, error)
	RenderPostHTML(ctx context.Context, qid string) (string, string, error)
	GetPostMarkdown(ctx context.Context, qid string) (string, string, error)
	GetUserPosts(ctx context.Context, userID int, offset, limit int) ([]post.Post, int64, error)
}

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

func RenderPost(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if c.Query("format") == "raw" {
			title, body, err := postSvc.GetPostMarkdown(c.Request.Context(), id)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}

			content := "# " + title + "\n\n" + body
			c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(content))
			return
		}

		title, htmlContent, err := postSvc.RenderPostHTML(c.Request.Context(), id)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
	}
}

func PostsList(postSvc PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUserPaginatedQuery(c, postSvc.GetUserPosts, newPostListItem,
			func(items []PostListItem, p Pagination) any {
				return PostsListResponse{Posts: items, Pagination: p}
			},
		)
	}
}
