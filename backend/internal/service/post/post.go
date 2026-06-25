// Package post provides post-related business logic and services.
package post

import (
	"bytes"
	"context"
	"regexp"

	"markpost/internal/domain/post"
	"markpost/internal/service"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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
	postRepo  post.Repository
	md        goldmark.Markdown
	sanitizer *bluemonday.Policy
	delivery  DeliveryEnqueuer
}

// NewService creates a new Service instance.
func NewService(postRepo post.Repository, delivery DeliveryEnqueuer) *Service {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithHardWraps(),
		),
	)

	return &Service{
		postRepo:  postRepo,
		md:        md,
		sanitizer: newPostHTMLSanitizer(),
		delivery:  delivery,
	}
}

// newPostHTMLSanitizer builds the HTML allowlist applied to every rendered post.
// UGCPolicy already permits GFM tables, <details>/<summary>, <del>, links and
// images while stripping <script>/<iframe>, event handlers and non-http(s)
// URL schemes. On top of it we allow the GFM tasklist checkbox and harden
// external links against tabnabbing.
func newPostHTMLSanitizer() *bluemonday.Policy {
	return bluemonday.UGCPolicy().
		AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input").
		AllowAttrs("checked", "disabled").OnElements("input").
		AddTargetBlankToFullyQualifiedLinks(true).
		RequireNoReferrerOnFullyQualifiedLinks(true)
}

// rawHTMLElementRe matches HTML elements parsed in raw-text / RCDATA mode.
// An unterminated one — e.g. a literal "<script>" that appears in post prose —
// makes the HTML tokenizer swallow the rest of the document as element text,
// hiding every following block (this is exactly the post1 bug, and it fools
// bluemonday the same way it fools a browser). Escaping only the opening "<"
// renders such tags as inert visible text before sanitization, so the page can
// no longer be truncated and the sanitizer keeps full control.
var rawHTMLElementRe = regexp.MustCompile(
	`(?i)<(/?)(script|style|iframe|noscript|noframes|noembed|textarea|title|xmp|plaintext)([\s/>])`,
)

func neutralizeRawHTMLElements(htmlContent string) string {
	return rawHTMLElementRe.ReplaceAllString(htmlContent, "&lt;$1$2$3")
}

func (s *Service) getPostByQID(ctx context.Context, qid string) (*post.Post, error) {
	p, err := s.postRepo.GetByQID(ctx, qid)
	if err != nil {
		return nil, service.WrapNotFoundOrInternal(err, "post not found", "get post failed")
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

	return p.Title, s.sanitizer.Sanitize(neutralizeRawHTMLElements(buf.String())), nil
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
	return service.Paginate(
		func() ([]post.Post, error) { return s.postRepo.GetByUserID(ctx, userID, offset, limit) },
		func() (int64, error) { return s.postRepo.CountByUserID(ctx, userID) },
		"user posts",
	)
}

// GetAllPosts retrieves all posts with optional search and pagination.
func (s *Service) GetAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error) {
	return service.Paginate(
		func() ([]post.Post, error) { return s.postRepo.ListAll(ctx, search, offset, limit) },
		func() (int64, error) { return s.postRepo.CountAll(ctx, search) },
		"all posts",
	)
}

// UpdatePost updates a post's title and body.
func (s *Service) UpdatePost(ctx context.Context, id int, title, body string) error {
	return service.WrapNotFoundOrInternal(s.postRepo.UpdateByID(ctx, id, title, body), "post not found", "update post failed")
}

// DeletePost deletes a post by its ID.
func (s *Service) DeletePost(ctx context.Context, id int) error {
	_, err := s.postRepo.DeleteByID(ctx, id)
	return service.WrapNotFoundOrInternal(err, "post not found", "delete post failed")
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
