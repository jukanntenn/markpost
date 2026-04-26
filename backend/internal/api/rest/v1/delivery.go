// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"
	"strconv"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// DeliveryService provides delivery channel operations.
type DeliveryService interface {
	ListByUserID(ctx context.Context, userID int) ([]delivery.Channel, error)
	Create(ctx context.Context, userID int, kind string, name string, webhookURL string, keywords string) (*delivery.Channel, error)
	Update(ctx context.Context, userID int, id int, kind string, name string, webhookURL string, keywords string, enabled *bool) (*delivery.Channel, error)
	Delete(ctx context.Context, userID int, id int) error
}

// ListDeliveryChannels returns a handler for listing user's delivery channels.
func ListDeliveryChannels(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

		channels, err := deliverySvc.ListByUserID(c.Request.Context(), u.ID)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := make([]gin.H, 0, len(channels))
		for _, ch := range channels {
			items = append(items, channelToJSON(ch))
		}

		c.JSON(http.StatusOK, gin.H{"channels": items})
	}
}

// CreateDeliveryChannelRequest represents a delivery channel creation request.
type CreateDeliveryChannelRequest struct {
	Kind       string `json:"kind" binding:"required"`
	Name       string `json:"name" binding:"required"`
	WebhookURL string `json:"webhook_url" binding:"required"`
	Keywords   string `json:"keywords"`
}

// CreateDeliveryChannel returns a handler for creating a delivery channel.
func CreateDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

		var req CreateDeliveryChannelRequest
		if !bindJSON(c, &req) {
			return
		}

		ch, err := deliverySvc.Create(c.Request.Context(), u.ID, req.Kind, req.Name, req.WebhookURL, req.Keywords)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"channel": channelToJSON(*ch)})
	}
}

// UpdateDeliveryChannelRequest represents a delivery channel update request.
type UpdateDeliveryChannelRequest struct {
	Kind       *string `json:"kind"`
	Name       *string `json:"name"`
	WebhookURL *string `json:"webhook_url"`
	Keywords   *string `json:"keywords"`
	Enabled    *bool   `json:"enabled"`
}

// UpdateDeliveryChannel returns a handler for updating a delivery channel.
func UpdateDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			apierr.RespondError(c, service.NewServiceError(service.ErrValidation, "invalid channel ID"))
			return
		}

		var req UpdateDeliveryChannelRequest
		if !bindJSON(c, &req) {
			return
		}

		var kind, name, webhookURL, keywords string
		if req.Kind != nil {
			kind = *req.Kind
		}
		if req.Name != nil {
			name = *req.Name
		}
		if req.WebhookURL != nil {
			webhookURL = *req.WebhookURL
		}
		if req.Keywords != nil {
			keywords = *req.Keywords
		}

		ch, err := deliverySvc.Update(c.Request.Context(), u.ID, id, kind, name, webhookURL, keywords, req.Enabled)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"channel": channelToJSON(*ch)})
	}
}

// DeleteDeliveryChannel returns a handler for deleting a delivery channel.
func DeleteDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			apierr.RespondError(c, service.NewServiceError(service.ErrValidation, "invalid channel ID"))
			return
		}

		if err := deliverySvc.Delete(c.Request.Context(), u.ID, id); err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
	}
}

func channelToJSON(ch delivery.Channel) gin.H {
	return gin.H{
		"id":         ch.ID,
		"kind":       ch.Kind,
		"name":       ch.Name,
		"enabled":    ch.Enabled,
		"webhook_url": ch.WebhookURL,
		"keywords":   ch.Keywords,
		"created_at": ch.CreatedAt,
		"updated_at": ch.UpdatedAt,
	}
}
