package v1

import (
	"context"
	"net/http"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/user"
	delivery_svc "markpost/internal/service/delivery"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// DeliveryService defines the interface for delivery channel operations.
type DeliveryService interface {
	ListByUserID(ctx context.Context, userID int) ([]delivery.Channel, error)
	Create(ctx context.Context, userID int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error)
	Update(ctx context.Context, userID int, id int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error)
	Delete(ctx context.Context, userID int, id int) error
}

// ListDeliveryChannels returns a handler that lists all delivery channels for the authenticated user.
func ListDeliveryChannels(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			channels, err := deliverySvc.ListByUserID(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}

			writeList(c, channels, newChannelResponse,
				func(items []ChannelResponse) any {
					return ChannelsListResponse{Channels: items}
				},
			)
		})
	}
}

// CreateDeliveryChannel returns a handler that creates a new delivery channel for the authenticated user.
func CreateDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			var req CreateDeliveryChannelRequest
			if !bindJSON(c, &req) {
				return
			}

			ch, err := deliverySvc.Create(c.Request.Context(), u.ID, req.toParams())
			if err != nil {
				apierr.RespondError(c, err)
				return
			}

			c.JSON(http.StatusCreated, SingleChannelResponse{Channel: newChannelResponse(*ch)})
		})
	}
}

// UpdateDeliveryChannel returns a handler that updates an existing delivery channel for the authenticated user.
func UpdateDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUserAndID(c, func(u *user.User, id int) {
			var req UpdateDeliveryChannelRequest
			if !bindJSON(c, &req) {
				return
			}

			ch, err := deliverySvc.Update(c.Request.Context(), u.ID, id, req.toParams())
			if err != nil {
				apierr.RespondError(c, err)
				return
			}

			c.JSON(http.StatusOK, SingleChannelResponse{Channel: newChannelResponse(*ch)})
		})
	}
}

// DeleteDeliveryChannel returns a handler that deletes an existing delivery channel for the authenticated user.
func DeleteDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUserAndID(c, func(u *user.User, id int) {
			if err := deliverySvc.Delete(c.Request.Context(), u.ID, id); err != nil {
				apierr.RespondError(c, err)
				return
			}

			c.JSON(http.StatusOK, MessageResponse{
				Message: getI18nMessage(c, "Channel deleted successfully", "delivery.channel_deleted"),
			})
		})
	}
}
