package v1

import (
	"context"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"

	"github.com/gin-gonic/gin"
)

type AdminService interface {
	ListAllUsers(ctx context.Context, offset, limit int) ([]user.User, int64, error)
	ListAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error)
	ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
}

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
