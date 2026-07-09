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
	ListHistory(ctx context.Context, userID, offset, limit int) ([]*delivery.HistoryRow, int64, error)
}

// ListDeliveryChannels godoc
// @Summary List the current user's delivery channels
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ChannelsListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/channels [get]
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

// CreateDeliveryChannel godoc
// @Summary Create a delivery channel
// @Tags delivery
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateDeliveryChannelRequest true "Channel kind, name, configuration and keywords"
// @Success 201 {object} SingleChannelResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/channels [post]
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

// UpdateDeliveryChannel godoc
// @Summary Update a delivery channel
// @Tags delivery
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param body body UpdateDeliveryChannelRequest true "Channel fields to update"
// @Success 200 {object} SingleChannelResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/channels/{id} [put]
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

// DeleteDeliveryChannel godoc
// @Summary Delete a delivery channel
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} MessageResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/channels/{id} [delete]
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

// ListDeliveryHistory godoc
// @Summary List the current user's delivery history
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.DeliveryHistoryListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/history [get]
func ListDeliveryHistory(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUserPaginatedQuery(c,
			deliverySvc.ListHistory,
			newDeliveryHistoryItem,
			paginatedWrap[DeliveryHistoryItem]("history"),
		)
	}
}
