package v1

import (
	"context"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	_ "markpost/internal/apierr"

	"github.com/gin-gonic/gin"
)

// AdminService defines the interface for admin-related operations.
type AdminService interface {
	ListAllUsers(ctx context.Context, offset, limit int) ([]user.User, int64, error)
	ListAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error)
	ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
	ListAllDeliveryHistory(ctx context.Context, offset, limit int) ([]*delivery.HistoryRow, int64, error)
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
