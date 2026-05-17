// Package post provides post-related business logic and services.
package post

import (
	"bytes"
	"context"
	"errors"

	"markpost/internal/domain/post"
	"markpost/internal/service"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

// DeliveryEnqueuer defines the interface for enqueueing delivery jobs.
type DeliveryEnqueuer interface {
	Enqueue(job DeliveryJob)
}

// DeliveryJob represents a delivery job.
type DeliveryJob struct {
	UserID  int
	PostQID string
	Title   string
	Body    string
}

// Service provides post-related business logic.
type Service struct {
	postRepo post.Repository
	md       goldmark.Markdown
	delivery DeliveryEnqueuer
}

// NewService creates a new Service instance.
func NewService(postRepo post.Repository, delivery DeliveryEnqueuer) *Service {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	return &Service{
		postRepo: postRepo,
		md:       md,
		delivery: delivery,
	}
}

func (s *Service) getPostByQID(ctx context.Context, qid string) (*post.Post, error) {
	p, err := s.postRepo.GetByQID(ctx, qid)
	if err != nil {
		if errors.Is(err, post.ErrNotFound) {
			return nil, service.NewServiceErrorWrap(service.ErrNotFound, "post not found", err)
		}
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "get post failed", err)
	}
	return p, nil
}

// CreatePost creates a new post and enqueues it for delivery.
func (s *Service) CreatePost(ctx context.Context, title, body string, userID int) (string, error) {
	p, err := s.postRepo.Create(ctx, title, body, userID)
	if err != nil {
		return "", service.NewServiceErrorWrap(service.ErrInternal, "create post failed", err)
	}

	if s.delivery != nil {
		s.delivery.Enqueue(DeliveryJob{
			UserID:  userID,
			PostQID: p.QID,
			Title:   title,
			Body:    body,
		})
	}

	return p.QID, nil
}

// RenderPostHTML renders a post's body as HTML.
func (s *Service) RenderPostHTML(ctx context.Context, qid string) (string, string, error) {
	p, err := s.getPostByQID(ctx, qid)
	if err != nil {
		return "", "", err
	}

	var buf bytes.Buffer
	if err := s.md.Convert([]byte(p.Body), &buf); err != nil {
		return "", "", service.NewServiceErrorWrap(service.ErrInternal, "render post failed", err)
	}

	return p.Title, buf.String(), nil
}

// GetPostMarkdown retrieves a post's raw markdown content.
func (s *Service) GetPostMarkdown(ctx context.Context, qid string) (string, string, error) {
	p, err := s.getPostByQID(ctx, qid)
	if err != nil {
		return "", "", err
	}

	return p.Title, p.Body, nil
}

// GetUserPosts retrieves posts for a specific user with pagination.
func (s *Service) GetUserPosts(ctx context.Context, userID int, offset, limit int) ([]post.Post, int64, error) {
	posts, err := s.postRepo.GetByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "get user posts failed", err)
	}

	total, err := s.postRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "count user posts failed", err)
	}

	return posts, total, nil
}

// GetAllPosts retrieves all posts with optional search and pagination.
func (s *Service) GetAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error) {
	posts, err := s.postRepo.ListAll(ctx, search, offset, limit)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "get all posts failed", err)
	}

	total, err := s.postRepo.CountAll(ctx, search)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "count all posts failed", err)
	}

	return posts, total, nil
}

// UpdatePost updates a post's title and body.
func (s *Service) UpdatePost(ctx context.Context, id int, title, body string) error {
	_, err := s.postRepo.UpdateByID(ctx, id, title, body)
	if err != nil {
		if errors.Is(err, post.ErrNotFound) {
			return service.NewServiceErrorWrap(service.ErrNotFound, "post not found", err)
		}
		return service.NewServiceErrorWrap(service.ErrInternal, "update post failed", err)
	}

	return nil
}

// DeletePost deletes a post by its ID.
func (s *Service) DeletePost(ctx context.Context, id int) error {
	_, err := s.postRepo.DeleteByID(ctx, id)
	if err != nil {
		return service.NewServiceErrorWrap(service.ErrInternal, "delete post failed", err)
	}

	return nil
}

// PruneExpired deletes expired posts based on retention days.
func (s *Service) PruneExpired(ctx context.Context, retentionDays, batchSize int) error {
	if retentionDays <= 0 {
		return service.NewServiceError(service.ErrValidation, "retention days must be positive")
	}
	if batchSize <= 0 {
		batchSize = 99
	}

	if err := s.postRepo.PruneExpired(ctx, retentionDays, batchSize); err != nil {
		return service.NewServiceErrorWrap(service.ErrInternal, "prune expired posts failed", err)
	}

	return nil
}

// CountExpired counts expired posts based on retention days.
func (s *Service) CountExpired(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, service.NewServiceError(service.ErrValidation, "retention days must be positive")
	}

	count, err := s.postRepo.CountExpired(ctx, retentionDays)
	if err != nil {
		return 0, service.NewServiceErrorWrap(service.ErrInternal, "count expired posts failed", err)
	}

	return count, nil
}
