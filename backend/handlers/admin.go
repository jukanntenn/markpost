package handlers

import (
	"net/http"
	"strconv"

	apperrors "markpost/errors"
	"markpost/models"
	"markpost/services"

	"github.com/gin-gonic/gin"
)

type AdminServiceInterface interface {
	UpdateUserRole(id int, role models.Role) (*models.User, error)
	DeleteUser(id int) error

	ListAllPosts(search string, offset int, limit int) ([]models.Post, int64, error)
	UpdatePost(id int, title string, body string) (*models.Post, error)
	DeletePost(id int) error

	ListAllDeliveryChannels() ([]models.DeliveryChannel, error)
	UpdateDeliveryChannel(id int, name *string, webhookURL *string, keywords *string, enabled *bool) (*models.DeliveryChannel, error)
	DeleteDeliveryChannel(id int) error
}

type AdminHandler struct {
	svc AdminServiceInterface
}

func NewAdminHandler(svc AdminServiceInterface) *AdminHandler {
	return &AdminHandler{svc: svc}
}

type updateUserRoleRequest struct {
	Role models.Role `json:"role" binding:"required,oneof=admin user"`
}

type adminUserResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	GitHubID  *int64 `json:"github_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: "id"},
		}))
		return
	}

	var req updateUserRoleRequest
	if !bindJSON(c, &req) {
		return
	}

	user, err := h.svc.UpdateUserRole(id, req.Role)
	if err != nil {
		apperrors.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": adminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Role:      string(user.Role),
			GitHubID:  user.GitHubID,
			CreatedAt: user.CreatedAt.Format(timeFormatRFC3339),
			UpdatedAt: user.UpdatedAt.Format(timeFormatRFC3339),
		},
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: "id"},
		}))
		return
	}

	if err := h.svc.DeleteUser(id); err != nil {
		apperrors.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type listAdminPostsQuery struct {
	Search string `form:"search"`
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type adminPostUserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type adminPostResponse struct {
	ID        int                    `json:"id"`
	QID       string                 `json:"qid"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	UserID    int                    `json:"user_id"`
	User      *adminPostUserResponse `json:"user,omitempty"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

func (h *AdminHandler) ListAllPosts(c *gin.Context) {
	var req listAdminPostsQuery
	if !bindQuery(c, &req) {
		return
	}

	page := defaultInt(req.Page, 1)
	limit := defaultInt(req.Limit, 10)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	posts, total, err := h.svc.ListAllPosts(req.Search, offset, limit)
	if err != nil {
		apperrors.RespondError(c, err)
		return
	}

	resp := make([]adminPostResponse, 0, len(posts))
	for _, p := range posts {
		var userResp *adminPostUserResponse
		if p.User.ID != 0 {
			userResp = &adminPostUserResponse{ID: p.User.ID, Username: p.User.Username}
		}
		resp = append(resp, adminPostResponse{
			ID:        p.ID,
			QID:       p.QID,
			Title:     p.Title,
			Body:      p.Body,
			UserID:    p.UserID,
			User:      userResp,
			CreatedAt: p.CreatedAt.Format(timeFormatRFC3339),
			UpdatedAt: p.UpdatedAt.Format(timeFormatRFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":     resp,
		"total":     total,
		"page":      page,
		"page_size": limit,
	})
}

type updatePostRequest struct {
	Title string `json:"title"`
	Body  string `json:"body" binding:"required"`
}

func (h *AdminHandler) UpdatePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: "id"},
		}))
		return
	}

	var req updatePostRequest
	if !bindJSON(c, &req) {
		return
	}

	post, err := h.svc.UpdatePost(id, req.Title, req.Body)
	if err != nil {
		apperrors.RespondError(c, err)
		return
	}

	var userResp *adminPostUserResponse
	if post.User.ID != 0 {
		userResp = &adminPostUserResponse{ID: post.User.ID, Username: post.User.Username}
	}

	c.JSON(http.StatusOK, gin.H{
		"post": adminPostResponse{
			ID:        post.ID,
			QID:       post.QID,
			Title:     post.Title,
			Body:      post.Body,
			UserID:    post.UserID,
			User:      userResp,
			CreatedAt: post.CreatedAt.Format(timeFormatRFC3339),
			UpdatedAt: post.UpdatedAt.Format(timeFormatRFC3339),
		},
	})
}

func (h *AdminHandler) DeletePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: "id"},
		}))
		return
	}

	if err := h.svc.DeletePost(id); err != nil {
		apperrors.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type adminDeliveryChannelResponse struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
	Keywords   string `json:"keywords"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func (h *AdminHandler) ListAllDeliveryChannels(c *gin.Context) {
	channels, err := h.svc.ListAllDeliveryChannels()
	if err != nil {
		apperrors.RespondError(c, err)
		return
	}

	resp := make([]adminDeliveryChannelResponse, 0, len(channels))
	for _, ch := range channels {
		resp = append(resp, adminDeliveryChannelResponse{
			ID:         ch.ID,
			UserID:     ch.UserID,
			Username:   ch.User.Username,
			Kind:       string(ch.Kind),
			Name:       ch.Name,
			Enabled:    ch.Enabled,
			WebhookURL: ch.WebhookURL,
			Keywords:   ch.Keywords,
			CreatedAt:  ch.CreatedAt.Format(timeFormatRFC3339),
			UpdatedAt:  ch.UpdatedAt.Format(timeFormatRFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"channels": resp})
}

func (h *AdminHandler) UpdateDeliveryChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: "id"},
		}))
		return
	}

	var req updateDeliveryChannelRequest
	if !bindJSON(c, &req) {
		return
	}

	if req.Name == nil && req.Enabled == nil && req.WebhookURL == nil && req.Keywords == nil {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: ""},
		}))
		return
	}

	channel, err := h.svc.UpdateDeliveryChannel(id, req.Name, req.WebhookURL, req.Keywords, req.Enabled)
	if err != nil {
		apperrors.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channel": adminDeliveryChannelResponse{
			ID:         channel.ID,
			UserID:     channel.UserID,
			Username:   channel.User.Username,
			Kind:       string(channel.Kind),
			Name:       channel.Name,
			Enabled:    channel.Enabled,
			WebhookURL: channel.WebhookURL,
			Keywords:   channel.Keywords,
			CreatedAt:  channel.CreatedAt.Format(timeFormatRFC3339),
			UpdatedAt:  channel.UpdatedAt.Format(timeFormatRFC3339),
		},
	})
}

func (h *AdminHandler) DeleteDeliveryChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
			{Code: services.ErrFieldViolation, Description: "id"},
		}))
		return
	}

	if err := h.svc.DeleteDeliveryChannel(id); err != nil {
		apperrors.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
