package post

import (
	"bytes"
	"context"
	"fmt"

	"markpost/internal/domain/post"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

type DeliveryEnqueuer interface {
	Enqueue(job DeliveryJob)
}

type DeliveryJob struct {
	UserID  int
	PostQID string
	Title   string
	Body    string
}

type PostService struct {
	postRepo post.Repository
	md       goldmark.Markdown
	delivery DeliveryEnqueuer
}

func NewPostService(postRepo post.Repository, delivery DeliveryEnqueuer) *PostService {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	return &PostService{
		postRepo: postRepo,
		md:       md,
		delivery: delivery,
	}
}

func (s *PostService) CreatePost(ctx context.Context, title, body string, userID int) (string, error) {
	p, err := s.postRepo.Create(ctx, title, body, userID)
	if err != nil {
		return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
	}
	if s.delivery != nil {
		s.delivery.Enqueue(DeliveryJob{
			UserID:  userID,
			PostQID: p.QID,
			Title:  p.Title,
			Body:    p.Body,
		})
	}
	return p.QID, nil
}

func (s *PostService) RenderPostHTML(ctx context.Context, qid string) (string, string, error) {
	p, err := s.postRepo.GetByQID(ctx, qid)
	if err != nil {
		return "", "", NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("post with qid %s not found", qid), err)
	}

	var buf bytes.Buffer
	if err := s.md.Convert([]byte(p.Body), &buf); err != nil {
		return "", "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("convert post with qid %s failed", qid), err)
	}

	return p.Title, buf.String(), nil
}

func (s *PostService) GetPostMarkdown(ctx context.Context, qid string) (string, string, error) {
	p, err := s.postRepo.GetByQID(ctx, qid)
	if err != nil {
		return "", "", NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("post with qid %s not found", qid), err)
	}

	return p.Title, p.Body, nil
}

func (s *PostService) GetUserPosts(ctx context.Context, userID int, page int, limit int) ([]post.Post, int64, error) {
	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 20
	}

	total, err := s.postRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("query posts count failed with userID %d", userID), err)
	}

	offset := (page - 1) * limit
	posts, err := s.postRepo.GetByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("query posts failed with userID %d", userID), err)
	}

	return posts, total, nil
}
