package handlers

import (
	"net/http"
	"strconv"

	apperrors "markpost/errors"
	"markpost/models"
	"markpost/services"

	"github.com/gin-gonic/gin"
)

type DeliveryChannelServiceInterface interface {
	ListByUserID(userID int) ([]models.DeliveryChannel, error)
	Create(userID int, kind models.DeliveryChannelKind, name string, webhookURL string, keywords string, enabled bool) (*models.DeliveryChannel, error)
	Update(userID int, id int, name *string, webhookURL *string, keywords *string, enabled *bool) (*models.DeliveryChannel, error)
	Delete(userID int, id int) error
}

type DeliveryChannelResponse struct {
	ID         int    `json:"id"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
	Keywords   string `json:"keywords"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func ListDeliveryChannels(svc DeliveryChannelServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil))
			return
		}

		channels, err := svc.ListByUserID(user.ID)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}

		resp := make([]DeliveryChannelResponse, 0, len(channels))
		for _, ch := range channels {
			resp = append(resp, DeliveryChannelResponse{
				ID:         ch.ID,
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
}

type createDeliveryChannelRequest struct {
	Kind       string `json:"kind" binding:"required"`
	Name       string `json:"name"`
	Enabled    *bool  `json:"enabled"`
	WebhookURL string `json:"webhook_url" binding:"required"`
	Keywords   string `json:"keywords"`
}

func CreateDeliveryChannel(svc DeliveryChannelServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil))
			return
		}

		var req createDeliveryChannelRequest
		if !bindJSON(c, &req) {
			return
		}

		kind, err := services.ParseDeliveryChannelKind(req.Kind)
		if err != nil {
			apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
				{Code: services.ErrFieldViolation, Description: "kind"},
			}))
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		channel, err := svc.Create(user.ID, kind, req.Name, req.WebhookURL, req.Keywords, enabled)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"channel": DeliveryChannelResponse{
				ID:         channel.ID,
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
}

type updateDeliveryChannelRequest struct {
	Name       *string `json:"name"`
	Enabled    *bool   `json:"enabled"`
	WebhookURL *string `json:"webhook_url"`
	Keywords   *string `json:"keywords"`
}

func UpdateDeliveryChannel(svc DeliveryChannelServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil))
			return
		}

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

		channel, err := svc.Update(user.ID, id, req.Name, req.WebhookURL, req.Keywords, req.Enabled)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"channel": DeliveryChannelResponse{
				ID:         channel.ID,
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
}

func DeleteDeliveryChannel(svc DeliveryChannelServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil))
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
				{Code: services.ErrFieldViolation, Description: "id"},
			}))
			return
		}

		if err := svc.Delete(user.ID, id); err != nil {
			apperrors.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"
