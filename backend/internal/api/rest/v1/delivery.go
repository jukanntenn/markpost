// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
	delivery_svc "markpost/internal/service/delivery"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// DeliveryService provides delivery channel operations.
type DeliveryService interface {
	ListByUserID(ctx context.Context, userID int) ([]delivery.Channel, error)
	Create(ctx context.Context, userID int, params delivery_svc.CreateChannelParams) (*delivery.Channel, error)
	Update(ctx context.Context, userID int, id int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error)
	Delete(ctx context.Context, userID int, id int) error
}

// ChannelResponse is the typed JSON response for a delivery channel.
type ChannelResponse struct {
	ID         int                  `json:"id"`
	Kind       delivery.ChannelKind `json:"kind"`
	Name       string               `json:"name"`
	Enabled    bool                 `json:"enabled"`
	WebhookURL string               `json:"webhook_url"`
	Keywords   string               `json:"keywords"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

func newChannelResponse(ch delivery.Channel) ChannelResponse {
	return ChannelResponse{
		ID:         ch.ID,
		Kind:       ch.Kind,
		Name:       ch.Name,
		Enabled:    ch.Enabled,
		WebhookURL: ch.WebhookURL,
		Keywords:   ch.Keywords,
		CreatedAt:  ch.CreatedAt,
		UpdatedAt:  ch.UpdatedAt,
	}
}

type ChannelsListResponse struct {
	Channels []ChannelResponse `json:"channels"`
}

type SingleChannelResponse struct {
	Channel ChannelResponse `json:"channel"`
}

// ListDeliveryChannels returns a handler for listing user's delivery channels.
func ListDeliveryChannels(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
			return
		}

		channels, err := deliverySvc.ListByUserID(c.Request.Context(), u.ID)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		items := mapSlice(channels, newChannelResponse)

		c.JSON(http.StatusOK, ChannelsListResponse{Channels: items})
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
		u, ok := requireUser(c)
		if !ok {
			return
		}

		var req CreateDeliveryChannelRequest
		if !bindJSON(c, &req) {
			return
		}

		ch, err := deliverySvc.Create(c.Request.Context(), u.ID, delivery_svc.CreateChannelParams{
			Kind:       req.Kind,
			Name:       req.Name,
			WebhookURL: req.WebhookURL,
			Keywords:   req.Keywords,
		})
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusCreated, SingleChannelResponse{Channel: newChannelResponse(*ch)})
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
		u, ok := requireUser(c)
		if !ok {
			return
		}

		id, ok := parsePathID(c)
		if !ok {
			return
		}

		var req UpdateDeliveryChannelRequest
		if !bindJSON(c, &req) {
			return
		}

		ch, err := deliverySvc.Update(c.Request.Context(), u.ID, id, delivery_svc.UpdateChannelParams{
			Kind:       deref(req.Kind),
			Name:       deref(req.Name),
			WebhookURL: deref(req.WebhookURL),
			Keywords:   deref(req.Keywords),
			Enabled:    req.Enabled,
		})
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, SingleChannelResponse{Channel: newChannelResponse(*ch)})
	}
}

// DeleteDeliveryChannel returns a handler for deleting a delivery channel.
func DeleteDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
			return
		}

		id, ok := parsePathID(c)
		if !ok {
			return
		}

		if err := deliverySvc.Delete(c.Request.Context(), u.ID, id); err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, MessageResponse{
			Message: getI18nMessage(c, "Channel deleted successfully", "delivery.channel_deleted"),
		})
	}
}

func parsePathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		apierr.RespondError(c, service.NewServiceError(service.ErrValidation, "invalid ID"))
		return 0, false
	}
	return id, true
}
