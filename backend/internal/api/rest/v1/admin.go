package v1

import (
	"context"
	"fmt"
	"net/http"

	_ "markpost/internal/apierr"
	"markpost/internal/domain/audit"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"

	"github.com/gin-gonic/gin"
)

// AdminService defines the interface for admin-related operations.
type AdminService interface {
	ListAllUsers(ctx context.Context, offset, limit int) ([]user.User, int64, error)
	ListAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error)
	ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
	ListAllDeliveryHistory(ctx context.Context, offset, limit int) ([]*delivery.HistoryRow, int64, error)
	ListAuditLogs(ctx context.Context, offset, limit int) ([]audit.Log, int64, error)
	CreateUser(ctx context.Context, email, username, password string) (*user.User, error)
	SetUserRole(ctx context.Context, userID int, role user.Role) error
	ResetUserPassword(ctx context.Context, userID int, password string) error
	SetUserActive(ctx context.Context, userID int, active bool) error
	DeleteUser(ctx context.Context, userID int) (int64, error)
	GetUserByID(ctx context.Context, userID int) (*user.User, error)
	RecordAudit(ctx context.Context, e audit.Entry) error
}

// AdminListUsers godoc
// @Summary List all users (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedUsers
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/users [get]
func AdminListUsers(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		handlePaginatedQuery(c,
			bindPaginationQuery,
			adminSvc.ListAllUsers,
			newAdminUserItem,
			paginatedWrap[AdminUserItem]("users"),
		)
	}
}

// AdminListPosts godoc
// @Summary List all posts (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search keyword"
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedPosts
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/posts [get]
func AdminListPosts(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		handleSearchPaginatedQuery(c,
			bindAdminPostsQuery,
			adminSvc.ListAllPosts,
			newAdminPostItem,
			paginatedWrap[AdminPostItem]("posts"),
		)
	}
}

// AdminListChannels godoc
// @Summary List all delivery channels (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedChannels
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/channels [get]
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
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.DeliveryHistoryListResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/delivery-history [get]
func AdminListDeliveryHistory(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		handlePaginatedQuery(c,
			bindPaginationQuery,
			adminSvc.ListAllDeliveryHistory,
			newDeliveryHistoryItem,
			paginatedWrap[DeliveryHistoryItem]("history"),
		)
	}
}

// AdminListAuditLogs godoc
// @Summary List audit logs (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (min 1)" default(1)
// @Param limit query int false "Items per page (min 1)" default(20)
// @Success 200 {object} v1.PaginatedAuditLogs
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 403 {object} apierr.ErrorResponse
// @Router /api/v1/admin/audit-logs [get]
func AdminListAuditLogs(adminSvc AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		handlePaginatedQuery(c,
			bindPaginationQuery,
			adminSvc.ListAuditLogs,
			newAdminAuditLogItem,
			paginatedWrap[AdminAuditLogItem]("audit_logs"),
		)
	}
}

// AdminCreateUserRequest represents the request body for creating a user (admin).
type AdminCreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
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
		if err := c.ShouldBindJSON(&req); err != nil {
			respondValidationError(c, err)
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
		if err := c.ShouldBindJSON(&req); err != nil {
			respondValidationError(c, err)
			return
		}

		if err := adminSvc.SetUserRole(c.Request.Context(), userID, user.Role(req.Role)); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.set_role",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			Metadata:   map[string]any{"role": req.Role},
		})

		u, err := adminSvc.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, newAdminUserItem(*u))
	}
}

// AdminResetUserPasswordRequest represents the request body for resetting a user's password (admin).
type AdminResetUserPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// AdminResetUserPassword godoc
// @Summary Reset a user's password (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param body body AdminResetUserPasswordRequest true "New password"
// @Success 200 {object} v1.AdminUserItem
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

		var req AdminResetUserPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondValidationError(c, err)
			return
		}

		if err := adminSvc.ResetUserPassword(c.Request.Context(), userID, req.Password); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.reset_password",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
		})

		u, err := adminSvc.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, newAdminUserItem(*u))
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
		if err := c.ShouldBindJSON(&req); err != nil {
			respondValidationError(c, err)
			return
		}

		if err := adminSvc.SetUserActive(c.Request.Context(), userID, req.Active); err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.set_active",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			Metadata:   map[string]any{"active": req.Active},
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

		deleted, err := adminSvc.DeleteUser(c.Request.Context(), userID)
		if err != nil {
			respondError(c, err)
			return
		}

		_ = adminSvc.RecordAudit(c.Request.Context(), audit.Entry{
			ActorID:    currentUserID(c),
			Action:     "user.delete",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
		})

		c.JSON(http.StatusOK, gin.H{"deleted": deleted})
	}
}
