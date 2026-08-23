package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	_ "markpost/internal/apierr"
	"markpost/internal/domain/audit"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/settings"
	"markpost/internal/domain/user"
	"markpost/internal/service"
	adminsvc "markpost/internal/service/admin"

	"github.com/gin-gonic/gin"
)

// AdminService defines the interface for admin-related operations.
type AdminService interface {
	ListAllUsers(ctx context.Context, search string, offset, limit int) ([]user.User, int64, error)
	ListAllPosts(ctx context.Context, search, username string, offset, limit int) ([]post.Post, int64, error)
	ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
	ListAllDeliveryHistory(ctx context.Context, filter delivery.HistoryFilter, offset, limit int) ([]*delivery.HistoryRow, int64, error)
	ListAuditLogs(ctx context.Context, filter audit.AuditFilter, offset, limit int) ([]audit.LogRow, int64, error)
	AuditActionCounts(ctx context.Context, filter audit.AuditFilter) (map[string]int64, error)
	CreateUser(ctx context.Context, email, username, password string) (*user.User, error)
	SetUserRole(ctx context.Context, actorID, userID int, role user.Role) error
	ResetUserPassword(ctx context.Context, userID int) (string, error)
	SetUserActive(ctx context.Context, actorID, userID int, active bool) error
	SetUserVIP(ctx context.Context, userID int, vip bool) error
	DeleteUser(ctx context.Context, actorID, userID int) (int64, error)
	GetUserByID(ctx context.Context, userID int) (*user.User, error)
	CreateChannel(ctx context.Context, channel *delivery.Channel) error
	GetChannelByID(ctx context.Context, id int, userID int) (*delivery.Channel, error)
	UpdateChannel(ctx context.Context, channel *delivery.Channel) error
	SetChannelEnabled(ctx context.Context, id int, enabled bool) error
	DeleteChannel(ctx context.Context, id int, userID int) (int64, error)
	DeleteChannelByID(ctx context.Context, id int) (int64, error)
	ListUserSessions(ctx context.Context, userID int) ([]user.RefreshToken, error)
	RevokeUserSessions(ctx context.Context, userID int) error
	RevokeSessionByID(ctx context.Context, tokenID int) error
	GetSettings(ctx context.Context) ([]settings.Setting, error)
	SetSetting(ctx context.Context, actorID int, key string, value settings.SettingValue) error
	GetStats(ctx context.Context) (*adminsvc.Stats, error)
	DailyStatsAll(ctx context.Context, days int) ([]*delivery.DailyStat, error)
	LockedChannels(ctx context.Context) ([]*delivery.LockedChannel, error)
	RecordAudit(ctx context.Context, e audit.Entry) error
}

// AdminListUsers godoc
// @Summary List all users (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param search query string false "Username search (LIKE)"
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedItemsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users [get]
func AdminListUsers(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q AdminUsersQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			writeBindingError(c, &q, err)
			return
		}
		if !validatePaginationQuery(c, &q.PaginationQuery) {
			return
		}
		items, total, err := adminSvc.ListAllUsers(c.Request.Context(), q.Search, q.Offset, q.Limit)
		if err != nil {
			respondError(c, err)
			return
		}
		writePaginatedList(c, items, total, q.PaginationQuery, newAdminUserItem, paginatedWrap[AdminUserItem]("users"))
	}
}

// AdminGetUser godoc
// @Summary Get a single user (admin) — detail profile data (D3.2)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} v1.AdminUserItem
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id} [get]
func AdminGetUser(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}
		u, err := adminSvc.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, newAdminUserItem(*u))
	}
}

// AdminListPosts godoc
// @Summary List all posts (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search keyword"
// @Param username query string false "Username filter (F.9)"
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedItemsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/posts [get]
func AdminListPosts(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q AdminPostsQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			writeBindingError(c, &q, err)
			return
		}
		if !validatePaginationQuery(c, &q.PaginationQuery) {
			return
		}
		items, total, err := adminSvc.ListAllPosts(c.Request.Context(), q.Search, q.Username, q.Offset, q.Limit)
		if err != nil {
			respondError(c, err)
			return
		}
		writePaginatedList(c, items, total, q.PaginationQuery, newAdminPostItem, paginatedWrap[AdminPostItem]("posts"))
	}
}

// AdminListChannels godoc
// @Summary List all delivery channels (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedItemsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery/channels [get]
func AdminListChannels(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		handlePaginatedQuery(c,
			bindPaginationQuery,
			adminSvc.ListAllDeliveryChannels,
			newAdminChannelItem,
			paginatedWrap[AdminChannelItem]("channels"),
		)
	}
}

// AdminListDeliveryHistory godoc
// @Summary List all delivery history (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param user_id query int false "User ID filter"
// @Param channel_id query int false "Channel ID filter"
// @Param status query string false "Status filter (delivered/failed/expired)"
// @Param error_category query string false "Error category filter (card_rejected/upstream_client_error/upstream_server_error/upstream_business_error/network/internal)"
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.DeliveryHistoryListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery/history [get]
func AdminListDeliveryHistory(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q AdminDeliveryHistoryQuery
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
		category, ok := parseHistoryErrorCategory(c, q.ErrorCategory)
		if !ok {
			return
		}
		filter := delivery.HistoryFilter{OwnerID: q.UserID, ChannelID: q.ChannelID, Status: status, ErrorCategory: category}
		items, total, err := adminSvc.ListAllDeliveryHistory(c.Request.Context(), filter, q.Offset, q.Limit)
		if err != nil {
			respondError(c, err)
			return
		}
		writePaginatedList(c, items, total, q.PaginationQuery, newDeliveryHistoryItem, paginatedWrap[DeliveryHistoryItem]("history"))
	}
}

// parseHistoryStatus maps the status query string to a delivery.Status,
// responding with a 422 on unknown values.
func parseHistoryStatus(c *gin.Context, s string) (delivery.Status, bool) {
	switch s {
	case "", "all":
		return 0, true
	case "delivered":
		return delivery.StatusDelivered, true
	case "failed":
		return delivery.StatusFailed, true
	case "expired":
		return delivery.StatusExpired, true
	}
	respondError(c, service.New(service.ErrInvalidRequest, "status must be one of: delivered, failed, expired"))
	return 0, false
}

// validDeliveryErrorCategories is the closed set of classified send-failure
// categories accepted by the admin delivery history error_category filter.
var validDeliveryErrorCategories = map[string]bool{
	"card_rejected":           true,
	"upstream_client_error":   true,
	"upstream_server_error":   true,
	"upstream_business_error": true,
	"network":                 true,
	"internal":                true,
}

// parseHistoryErrorCategory validates the error_category query string, returning
// the category (or "" for no filter) and responding 422 on unknown values.
func parseHistoryErrorCategory(c *gin.Context, s string) (string, bool) {
	switch s {
	case "", "all":
		return "", true
	}
	if validDeliveryErrorCategories[s] {
		return s, true
	}
	respondError(c, service.New(service.ErrInvalidRequest, "error_category must be one of: card_rejected, upstream_client_error, upstream_server_error, upstream_business_error, network, internal"))
	return "", false
}

// AdminListAuditLogs godoc
// @Summary List audit logs (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param actor_id query int false "Actor user ID filter"
// @Param action query string false "Action filter"
// @Param target_type query string false "Target type filter"
// @Param target_id query string false "Target ID filter"
// @Param since query string false "RFC3339 start time"
// @Param until query string false "RFC3339 end time"
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedItemsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/audit-logs [get]
func AdminListAuditLogs(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q AdminAuditQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			writeBindingError(c, &q, err)
			return
		}
		if !validatePaginationQuery(c, &q.PaginationQuery) {
			return
		}
		filter := audit.AuditFilter{
			ActorID:    q.ActorID,
			Action:     q.Action,
			TargetType: q.TargetType,
			TargetID:   q.TargetID,
		}
		if q.Since != "" {
			t, err := time.Parse(time.RFC3339, q.Since)
			if err != nil {
				respondError(c, service.New(service.ErrInvalidRequest, "since must be an RFC3339 timestamp"))
				return
			}
			filter.Since = &t
		}
		if q.Until != "" {
			t, err := time.Parse(time.RFC3339, q.Until)
			if err != nil {
				respondError(c, service.New(service.ErrInvalidRequest, "until must be an RFC3339 timestamp"))
				return
			}
			filter.Until = &t
		}

		rows, total, err := adminSvc.ListAuditLogs(c.Request.Context(), filter, q.Offset, q.Limit)
		if err != nil {
			respondError(c, err)
			return
		}
		facets, err := adminSvc.AuditActionCounts(c.Request.Context(), filter)
		if err != nil {
			respondError(c, err)
			return
		}
		items := make([]AdminAuditLogItem, len(rows))
		for i, row := range rows {
			items[i] = newAdminAuditLogItem(row)
		}
		// I.10 统一契约：所有列表端点返回扁平 items envelope。
		c.JSON(http.StatusOK, gin.H{
			"items":       items,
			"total":       int(total),
			"page":        q.Page,
			"limit":       q.Limit,
			"total_pages": service.CalcTotalPages(total, q.Limit),
			"facets":      facets,
		})
	}
}

// AdminCreateUserRequest represents the request body for creating a user (admin).
// Password length policy is enforced by the service layer (C2.3).
type AdminCreateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AdminCreateUser godoc
// @Summary Create a user (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body AdminCreateUserRequest true "User details"
// @Success 201 {object} v1.AdminUserItem
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users [post]
func AdminCreateUser(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AdminCreateUserRequest
		if !bindJSON(c, &req) {
			return
		}

		u, err := adminSvc.CreateUser(c.Request.Context(), req.Email, req.Username, req.Password)
		if err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.create",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", u.ID),
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusCreated, newAdminUserItem(*u))
	}
}

// AdminSetUserRoleRequest represents the request body for setting a user's role (admin).
type AdminSetUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user"`
}

// AdminSetUserRole godoc
// @Summary Set a user's role (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param body body AdminSetUserRoleRequest true "New role"
// @Success 200 {object} v1.AdminUserItem
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id}/role [patch]
func AdminSetUserRole(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		var req AdminSetUserRoleRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := adminSvc.SetUserRole(c.Request.Context(), currentUserID(c), userID, user.Role(req.Role)); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.set_role",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			Metadata:   map[string]any{"role": req.Role},
			IP:         c.ClientIP(),
		})

		u, err := adminSvc.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, newAdminUserItem(*u))
	}
}

// AdminResetUserPassword godoc
// @Summary Reset a user's password (admin) — system generates a temporary
// password returned in plaintext exactly once (D3.3 方案 B)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} v1.AdminResetPasswordResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id}/password [post]
func AdminResetUserPassword(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		password, err := adminSvc.ResetUserPassword(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.reset_password",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusOK, AdminResetPasswordResponse{Password: password})
	}
}

// AdminSetUserActiveRequest represents the request body for setting a user's active status (admin).
type AdminSetUserActiveRequest struct {
	Active bool `json:"active"`
}

// AdminSetUserActive godoc
// @Summary Set a user's active status (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param body body AdminSetUserActiveRequest true "Active status"
// @Success 200 {object} v1.AdminUserItem
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id}/active [patch]
func AdminSetUserActive(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		var req AdminSetUserActiveRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := adminSvc.SetUserActive(c.Request.Context(), currentUserID(c), userID, req.Active); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.set_active",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			Metadata:   map[string]any{"active": req.Active},
			IP:         c.ClientIP(),
		})

		u, err := adminSvc.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, newAdminUserItem(*u))
	}
}

// AdminDeleteUser godoc
// @Summary Delete a user (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]int64
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id} [delete]
func AdminDeleteUser(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		deleted, err := adminSvc.DeleteUser(c.Request.Context(), currentUserID(c), userID)
		if err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.delete",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusOK, gin.H{"deleted": deleted})
	}
}

// AdminCreateChannelRequest represents the request body for creating a channel (admin).
type AdminCreateChannelRequest struct {
	UserID        int             `json:"user_id" binding:"required"`
	Kind          string          `json:"kind" binding:"required"`
	Name          string          `json:"name" binding:"required"`
	Configuration json.RawMessage `json:"configuration" binding:"required"`
	Keywords      string          `json:"keywords"`
}

// AdminCreateChannel godoc
// @Summary Create a delivery channel (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body AdminCreateChannelRequest true "Channel details"
// @Success 201 {object} v1.AdminChannelItem
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery/channels [post]
func AdminCreateChannel(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AdminCreateChannelRequest
		if !bindJSON(c, &req) {
			return
		}

		var config delivery.ChannelConfiguration
		if err := json.Unmarshal(req.Configuration, &config); err != nil {
			writeBindingError(c, &req, err)
			return
		}

		ch := &delivery.Channel{
			UserID:        req.UserID,
			Kind:          delivery.ChannelKind(req.Kind),
			Name:          req.Name,
			Configuration: config,
			Keywords:      req.Keywords,
			Enabled:       true,
		}

		if err := adminSvc.CreateChannel(c.Request.Context(), ch); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "channel.create",
			TargetType: "channel",
			TargetID:   fmt.Sprintf("%d", ch.ID),
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusCreated, newAdminChannelItem(*ch))
	}
}

// AdminSetChannelEnabledRequest represents the request body for enabling/disabling a channel.
type AdminSetChannelEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// AdminSetChannelEnabled godoc
// @Summary Enable or disable a delivery channel (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param body body AdminSetChannelEnabledRequest true "Enabled status"
// @Success 200 {object} v1.MessageResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery/channels/{id}/enabled [patch]
func AdminSetChannelEnabled(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		var req AdminSetChannelEnabledRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := adminSvc.SetChannelEnabled(c.Request.Context(), channelID, req.Enabled); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "channel.set_enabled",
			TargetType: "channel",
			TargetID:   fmt.Sprintf("%d", channelID),
			Metadata:   map[string]any{"enabled": req.Enabled},
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusOK, MessageResponse{Message: "Channel updated"})
	}
}

// AdminDeleteChannel godoc
// @Summary Delete a delivery channel (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} map[string]int64
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery/channels/{id} [delete]
func AdminDeleteChannel(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		deleted, err := adminSvc.DeleteChannelByID(c.Request.Context(), channelID)
		if err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "channel.delete",
			TargetType: "channel",
			TargetID:   fmt.Sprintf("%d", channelID),
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusOK, gin.H{"deleted": deleted})
	}
}

// AdminGetStats godoc
// @Summary Get admin dashboard statistics (D2.4, includes week deltas)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.AdminStatsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/stats [get]
func AdminGetStats(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := adminSvc.GetStats(c.Request.Context())
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"counts": stats})
	}
}

// AdminDeliveryStats godoc
// @Summary Get site-wide delivery trend (admin, D2.5)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param days query int false "Days to aggregate (default 7)"
// @Success 200 {object} v1.DeliveryStatsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery/stats [get]
func AdminDeliveryStats(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		days := 7
		if v := c.Query("days"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 || n > 365 {
				respondError(c, service.New(service.ErrInvalidRequest, "days must be a positive integer"))
				return
			}
			days = n
		}
		trend, err := adminSvc.DailyStatsAll(c.Request.Context(), days)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, DeliveryStatsResponse{Trend: trend})
	}
}

// AdminLockedChannels godoc
// @Summary List channels with persistent delivery failures (admin, D2.1)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.AdminLockedChannelsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/locked-channels [get]
func AdminLockedChannels(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := adminSvc.LockedChannels(c.Request.Context())
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, AdminLockedChannelsResponse{Items: items})
	}
}

// AdminListSessions godoc
// @Summary List user sessions (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} v1.SessionsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id}/sessions [get]
func AdminListSessions(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		tokens, err := adminSvc.ListUserSessions(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}

		c.JSON(http.StatusOK, SessionsResponse{Sessions: tokens})
	}
}

// AdminRevokeUserSessions godoc
// @Summary Revoke all user sessions (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]int64
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id}/sessions [delete]
func AdminRevokeUserSessions(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		if err := adminSvc.RevokeUserSessions(c.Request.Context(), userID); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.revoke_sessions",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			IP:         c.ClientIP(),
		})

		c.JSON(http.StatusOK, gin.H{"revoked": true})
	}
}

// AdminRevokeSession godoc
// @Summary Revoke a single user session (admin, D3.2)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param token_id path int true "Session (refresh token) ID"
// @Success 200 {object} map[string]bool
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/sessions/{token_id} [delete]
func AdminRevokeSession(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID, err := parseIDParam(c, "token_id")
		if err != nil {
			return
		}

		if err := adminSvc.RevokeSessionByID(c.Request.Context(), tokenID); err != nil {
			respondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"revoked": true})
	}
}

// AdminSettingItem represents one runtime setting row (admin).
type AdminSettingItem struct {
	Key       string                `json:"key"`
	Value     settings.SettingValue `json:"value"`
	UpdatedBy *int64                `json:"updated_by"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// AdminSettingListResponse wraps the settings rows (I.10: non-nil slice).
type AdminSettingListResponse struct {
	Items []AdminSettingItem `json:"items"`
}

func newAdminSettingItem(s settings.Setting) AdminSettingItem {
	return AdminSettingItem{
		Key:       s.Key,
		Value:     s.Value,
		UpdatedBy: s.UpdatedBy,
		UpdatedAt: s.UpdatedAt,
	}
}

// AdminSetSettingRequest represents the request body for upserting one
// runtime setting.
type AdminSetSettingRequest struct {
	Enabled bool `json:"enabled"`
}

// AdminGetSettings godoc
// @Summary List runtime settings (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.AdminSettingListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/settings [get]
func AdminGetSettings(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := adminSvc.GetSettings(c.Request.Context())
		if err != nil {
			respondError(c, err)
			return
		}
		items := make([]AdminSettingItem, 0, len(rows))
		for _, s := range rows {
			items = append(items, newAdminSettingItem(s))
		}
		c.JSON(http.StatusOK, AdminSettingListResponse{Items: items})
	}
}

// AdminSetSetting godoc
// @Summary Upsert one runtime setting (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key (v1: vip)"
// @Param body body AdminSetSettingRequest true "Setting value"
// @Success 200 {object} v1.AdminSettingListResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/settings/{key} [put]
func AdminSetSetting(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")

		var req AdminSetSettingRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := adminSvc.SetSetting(c.Request.Context(), currentUserID(c), key, settings.SettingValue{Enabled: req.Enabled}); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "setting.set",
			TargetType: "setting",
			TargetID:   key,
			Metadata:   map[string]any{"enabled": req.Enabled},
			IP:         c.ClientIP(),
		})

		rows, err := adminSvc.GetSettings(c.Request.Context())
		if err != nil {
			respondError(c, err)
			return
		}
		items := make([]AdminSettingItem, 0, len(rows))
		for _, s := range rows {
			items = append(items, newAdminSettingItem(s))
		}
		c.JSON(http.StatusOK, AdminSettingListResponse{Items: items})
	}
}

// AdminSetUserVIPRequest represents the request body for setting a user's VIP
// status (admin).
type AdminSetUserVIPRequest struct {
	VIP bool `json:"vip"`
}

// AdminSetUserVIP godoc
// @Summary Set a user's VIP status (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param body body AdminSetUserVIPRequest true "VIP status"
// @Success 200 {object} v1.AdminUserItem
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users/{id}/vip [patch]
func AdminSetUserVIP(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := parseIDParam(c, "id")
		if err != nil {
			return
		}

		var req AdminSetUserVIPRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := adminSvc.SetUserVIP(c.Request.Context(), userID, req.VIP); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.set_vip",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			Metadata:   map[string]any{"vip": req.VIP},
			IP:         c.ClientIP(),
		})

		u, err := adminSvc.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, newAdminUserItem(*u))
	}
}
