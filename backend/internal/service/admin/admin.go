// Package admin provides admin-level business logic and services.
package admin

import (
	"context"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"
)

// UserLister defines the interface for retrieving users.
type UserLister interface {
	GetAll(ctx context.Context, offset, limit int) ([]user.User, error)
	Count(ctx context.Context) (int64, error)
}

// PostLister defines the interface for retrieving posts.
type PostLister interface {
	GetAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error)
}

// ChannelLister defines the interface for retrieving delivery channels.
type ChannelLister interface {
	ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
}

// HistoryLister defines the interface for retrieving delivery history.
type HistoryLister interface {
	ListHistory(ctx context.Context, filter delivery.HistoryFilter, offset, limit int) ([]*delivery.HistoryRow, error)
	CountHistory(ctx context.Context, filter delivery.HistoryFilter) (int64, error)
}

// Service provides admin-level business logic.
type Service struct {
	userLister    UserLister
	postLister    PostLister
	channelLister ChannelLister
	historyLister HistoryLister
}

// NewService creates a new admin Service instance.
func NewService(userLister UserLister, postLister PostLister, channelLister ChannelLister, historyLister HistoryLister) *Service {
	return &Service{
		userLister:    userLister,
		postLister:    postLister,
		channelLister: channelLister,
		historyLister: historyLister,
	}
}

// ListAllUsers retrieves all users with pagination.
func (s *Service) ListAllUsers(ctx context.Context, offset, limit int) ([]user.User, int64, error) {
	return service.Paginate(
		func() ([]user.User, error) { return s.userLister.GetAll(ctx, offset, limit) },
		func() (int64, error) { return s.userLister.Count(ctx) },
		"users",
	)
}

// ListAllPosts retrieves all posts with optional search and pagination.
func (s *Service) ListAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error) {
	return s.postLister.GetAllPosts(ctx, search, offset, limit)
}

// ListAllDeliveryChannels retrieves all delivery channels with pagination.
func (s *Service) ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error) {
	return s.channelLister.ListAll(ctx, offset, limit)
}

// ListAllDeliveryHistory retrieves all delivery history (all users, including
// anonymized rows) with pagination.
func (s *Service) ListAllDeliveryHistory(ctx context.Context, offset, limit int) ([]*delivery.HistoryRow, int64, error) {
	filter := delivery.HistoryFilter{}
	return service.Paginate(
		func() ([]*delivery.HistoryRow, error) { return s.historyLister.ListHistory(ctx, filter, offset, limit) },
		func() (int64, error) { return s.historyLister.CountHistory(ctx, filter) },
		"delivery history",
	)
}
