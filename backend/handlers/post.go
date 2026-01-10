package handlers

import (
	"html/template"
	"net/http"

	apperrors "markpost/errors"
	"markpost/services"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type PostRequest struct {
	Title string `json:"title" binding:"required,titlesize"`
	Body  string `json:"body" binding:"required,bodysize"`
}

func CreatePost(postSvc *services.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			err := services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil)
			apperrors.RespondError(c, err)
			return
		}

		var req PostRequest
		if !bindJSON(c, &req) {
			return
		}

		id, err := postSvc.CreatePost(req.Title, req.Body, user.ID)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}

func RenderPost(postSvc *services.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		title, htmlContent, err := postSvc.RenderPostHTML(id)
		if err != nil {
			if se, ok := err.(*services.ServiceError); ok && se.Code == services.ErrNotFound {
				c.String(http.StatusNotFound, ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "error.not_found",
						Other: "Not Found",
					},
				}))
			} else {
				c.String(http.StatusInternalServerError, ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "error.failed_render_post",
						Other: "Failed to render post",
					},
				}))
			}
			return
		}
		c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
	}
}

func PostsList(postSvc *services.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			err := services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil)
			apperrors.RespondError(c, err)
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
			err := services.NewServiceErrorWrap(services.ErrInvalidRequest, "invalid limit", nil)
			apperrors.RespondError(c, err)
			return
		}

		posts, total, err := postSvc.GetUserPosts(user.ID, query.Page, query.Limit)
		if err != nil {
			apperrors.RespondError(c, err)
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
