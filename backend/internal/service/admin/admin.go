// Package admin provides admin-level business logic and services.
package admin

import (
	"context"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"
)

// UserRepository defines user data access for admin operations.
type UserRepository interface {
	GetAll(ctx context.Context, offset, limit int) ([]user.User, error)
	Count(ctx context.Context) (int64, error)
}

// PostRepository defines post data access for admin operations.
type PostRepository interface {
	ListAll(ctx context.Context, search string, offset int, limit int) ([]post.Post, error)
	CountAll(ctx context.Context, search string) (int64, error)
}

// DeliveryRepository defines delivery channel data access for admin operations.
type DeliveryRepository interface {
	ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, error)
	CountAll(ctx context.Context) (int64, error)
}

// Service provides admin-level business logic.
type Service struct {
	userRepo     UserRepository
	postRepo     PostRepository
	deliveryRepo DeliveryRepository
}

// NewService creates a new admin Service instance.
func NewService(userRepo UserRepository, postRepo PostRepository, deliveryRepo DeliveryRepository) *Service {
	return &Service{
		userRepo:     userRepo,
		postRepo:     postRepo,
		deliveryRepo: deliveryRepo,
	}
}

// ListAllUsers retrieves all users with pagination.
func (s *Service) ListAllUsers(ctx context.Context, page, limit int) ([]user.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, err := s.userRepo.GetAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "get users failed", err)
	}

	total, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "count users failed", err)
	}

	return users, total, nil
}

// ListAllPosts retrieves all posts with optional search and pagination.
func (s *Service) ListAllPosts(ctx context.Context, search string, page, limit int) ([]post.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	posts, err := s.postRepo.ListAll(ctx, search, offset, limit)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "get posts failed", err)
	}

	total, err := s.postRepo.CountAll(ctx, search)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "count posts failed", err)
	}

	return posts, total, nil
}

// ListAllDeliveryChannels retrieves all delivery channels with pagination.
func (s *Service) ListAllDeliveryChannels(ctx context.Context, page, limit int) ([]delivery.Channel, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	channels, err := s.deliveryRepo.ListAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "get channels failed", err)
	}

	total, err := s.deliveryRepo.CountAll(ctx)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "count channels failed", err)
	}

	return channels, total, nil
}
