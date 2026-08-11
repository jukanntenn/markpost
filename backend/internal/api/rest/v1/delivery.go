package v1

import (
	"context"
	"net/http"
	"strconv"

	"markpost/internal/apierr"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/user"
	"markpost/internal/service"
	delivery_svc "markpost/internal/service/delivery"

	"github.com/gin-gonic/gin"
)

// DeliveryService defines the interface for delivery channel operations.
type DeliveryService interface {
	ListByUserID(ctx context.Context, userID int) ([]delivery.Channel, error)
	Create(ctx context.Context, userID int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error)
	Update(ctx context.Context, userID int, id int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error)
	Delete(ctx context.Context, userID int, id int) error
	SendTest(ctx context.Context, userID, id int) error
	ListHistory(ctx context.Context, userID, channelID int, status delivery.Status, offset, limit int) ([]*delivery.HistoryRow, int64, error)
	LatestPerChannel(ctx context.Context, userID int) ([]*delivery.HistoryRow, error)
	PendingAttempts(ctx context.Context, userID int) ([]*delivery.PendingAttemptRow, error)
	DailyStats(ctx context.Context, userID, days int) ([]*delivery.DailyStat, error)
	TodayCounts(ctx context.Context, userID int) (*delivery.TodayCounts, error)
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
					return ChannelsListResponse{Items: items}
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

			// 204 No Content (no body) per api-design.md §2 (DELETE → 204).
			c.Status(http.StatusNoContent)
		})
	}
}

// TestDeliveryChannel godoc
// @Summary Send a test message to a delivery channel
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} MessageResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Failure 502 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/channels/{id}/test [post]
func TestDeliveryChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUserAndID(c, func(u *user.User, id int) {
			if err := deliverySvc.SendTest(c.Request.Context(), u.ID, id); err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, MessageResponse{Message: "test message sent"})
		})
	}
}

// LatestDeliveryPerChannel godoc
// @Summary List the most recent delivery per channel for the current user
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DeliveryLatestListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/latest [get]
func LatestDeliveryPerChannel(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			rows, err := deliverySvc.LatestPerChannel(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			writeList(c, rows, newDeliveryHistoryItem, func(items []DeliveryHistoryItem) any {
				return DeliveryLatestListResponse{Items: items}
			})
		})
	}
}

// ListDeliveryHistory godoc
// @Summary List the current user's delivery history
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Param channel_id query int false "Filter by delivery channel ID"
// @Param status query string false "Filter by terminal status (delivered/failed/expired)"
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.DeliveryHistoryListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/history [get]
func ListDeliveryHistory(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q DeliveryHistoryQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			writeBindingError(c, &q, err)
			return
		}
		if !validatePaginationQuery(c, &q.PaginationQuery) {
			return
		}
		status, ok := parseHistoryStatus(c, q.Status)
		if !ok {
			return
		}
		u, ok := requireUser(c)
		if !ok {
			return
		}
		items, total, err := deliverySvc.ListHistory(c.Request.Context(), u.ID, q.ChannelID, status, q.Offset, q.Limit)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		writePaginatedList(c, items, total, q.PaginationQuery, newDeliveryHistoryItem, paginatedWrap[DeliveryHistoryItem]("history"))
	}
}

// DeliveryStats godoc
// @Summary Get the current user's delivery trend + today counters (B2.7/K.2)
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Param days query int false "Days to aggregate (default 7)"
// @Success 200 {object} v1.DeliveryStatsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/stats [get]
func DeliveryStats(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		days := 7
		if v := c.Query("days"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 || n > 365 {
				apierr.RespondError(c, service.New(service.ErrInvalidRequest, "days must be a positive integer"))
				return
			}
			days = n
		}
		withUser(c, func(u *user.User) {
			today, err := deliverySvc.TodayCounts(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			trend, err := deliverySvc.DailyStats(c.Request.Context(), u.ID, days)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, DeliveryStatsResponse{Today: *today, Trend: trend})
		})
	}
}

// PendingDeliveryAttempts godoc
// @Summary List the current user's in-flight delivery attempts (K.2)
// @Tags delivery
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.PendingAttemptsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/delivery/pending [get]
func PendingDeliveryAttempts(deliverySvc DeliveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			rows, err := deliverySvc.PendingAttempts(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, PendingAttemptsResponse{Items: rows})
		})
	}
}
