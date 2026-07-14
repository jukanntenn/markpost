// Package post provides post-related business logic and services.
package post

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/post"
	"markpost/internal/service"

	"github.com/cespare/xxhash/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/tdewolff/minify/v2"
	minhtml "github.com/tdewolff/minify/v2/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/sync/singleflight"
)

// Service provides post-related business logic.
type Service struct {
	postRepo  post.Repository
	md        goldmark.Markdown
	sanitizer *bluemonday.Policy
	minifier  *minify.M
	delivery  post.DeliveryEnqueuer
	cache     renderCache
	group     singleflight.Group
	purger    Purger
}

// NewService creates a new Service instance. The in-process render cache
// (singleflight + ristretto) is built from the [render] config section; when
// disabled the service behaves exactly as it did before caching.
func NewService(postRepo post.Repository, delivery post.DeliveryEnqueuer) *Service {
	return &Service{
		postRepo:  postRepo,
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		delivery:  delivery,
		cache:     newRenderCache(),
		purger:    newPurger(),
	}
}

func newGoldmark() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithHardWraps(),
		),
	)
}

func newHTMLMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/html", minhtml.Minify)
	return m
}

// newRenderCache builds the render cache from config. A failure to construct
// ristretto is logged and degrades to a no-op cache rather than aborting
// startup, so a misconfigured cache never takes the read path down.
func newRenderCache() renderCache {
	cfg := config.Get().Render
	if !cfg.Enabled {
		return noopCache{}
	}
	maxCost := int64(cfg.CacheSizeBytes)
	if maxCost <= 0 {
		return noopCache{}
	}
	c, err := newRistrettoCache(maxCost, int64(cfg.NumCounters), int64(cfg.BufferItems))
	if err != nil {
		log.Printf("post: render cache disabled (ristretto init failed: %v)", err)
		return noopCache{}
	}
	return c
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
		return "", service.Wrap(service.ErrInternal, "create post failed", err)
	}

	if s.delivery != nil {
		s.delivery.Enqueue(post.DeliveryJob{
			UserID:  userID,
			PostID:  p.ID,
			PostQID: p.QID,
			Title:   title,
			Body:    body,
		})
	}

	return p.QID, nil
}

// RenderPostHTML renders a post's body as sanitized, minified HTML and returns
// the title, the rendered HTML, the response ETag (xxhash64 of the rendered
// output), and the post's creation time (for Last-Modified) for
// conditional-GET revalidation. The DB read + render pipeline runs behind a
// ristretto cache fronted by singleflight: a cache hit skips goldmark/bluemonday
// entirely, and concurrent misses for the same QID collapse to one render.
func (s *Service) RenderPostHTML(ctx context.Context, qid string) (title, html, etag string, createdAt time.Time, err error) {
	key := cacheKey(qid, "html")

	if v, ok := s.cache.Get(key); ok {
		return v.title, v.body, v.etag, v.createdAt, nil
	}

	v, err, _ := s.group.Do(key, func() (any, error) {
		if cached, ok := s.cache.Get(key); ok {
			return cached, nil
		}
		p, err := s.getPostByQID(ctx, qid)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := s.md.Convert([]byte(p.Body), &buf); err != nil {
			return nil, service.Wrap(service.ErrInternal, "render post failed", err)
		}
		sanitized := s.sanitizer.Sanitize(neutralizeRawHTMLElements(buf.String()))
		minified, mErr := s.minifyHTML(sanitized)
		if mErr != nil {
			return nil, service.Wrap(service.ErrInternal, "render post failed", mErr)
		}
		result := renderResult{
			title:     p.Title,
			body:      minified,
			etag:      etagHex(minified),
			createdAt: p.CreatedAt,
		}
		s.cache.Set(key, result, int64(len(minified)))
		return result, nil
	})
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	r, ok := v.(renderResult)
	if !ok {
		return "", "", "", time.Time{}, service.New(service.ErrInternal, "render post failed")
	}
	return r.title, r.body, r.etag, r.createdAt, nil
}

// GetPostMarkdown retrieves a post's raw markdown content and returns the
// title, the body, the response ETag (xxhash64 of the raw response body
// "# <title>\n\n<body>", matching what the handler serves), and the post's
// creation time (for Last-Modified). Like RenderPostHTML it is cache-fronted
// and singleflight-guarded, but its "render" is plain string concatenation
// (no goldmark/bluemonday), so the miss path is cheap.
func (s *Service) GetPostMarkdown(ctx context.Context, qid string) (title, body, etag string, createdAt time.Time, err error) {
	key := cacheKey(qid, "raw")

	if v, ok := s.cache.Get(key); ok {
		return v.title, v.body, v.etag, v.createdAt, nil
	}

	v, err, _ := s.group.Do(key, func() (any, error) {
		if cached, ok := s.cache.Get(key); ok {
			return cached, nil
		}
		p, err := s.getPostByQID(ctx, qid)
		if err != nil {
			return nil, err
		}
		rawBody := "# " + p.Title + "\n\n" + p.Body
		result := renderResult{
			title:     p.Title,
			body:      p.Body,
			etag:      etagHex(rawBody),
			createdAt: p.CreatedAt,
		}
		s.cache.Set(key, result, int64(len(rawBody)))
		return result, nil
	})
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	r, ok := v.(renderResult)
	if !ok {
		return "", "", "", time.Time{}, service.New(service.ErrInternal, "render post failed")
	}
	return r.title, r.body, r.etag, r.createdAt, nil
}

func (s *Service) minifyHTML(htmlContent string) (string, error) {
	var buf bytes.Buffer
	if err := s.minifier.Minify("text/html", &buf, strings.NewReader(htmlContent)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func etagHex(s string) string {
	return fmt.Sprintf("%016x", xxhash.Sum64String(s))
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

// DeletePost deletes a post by its ID. Prefer DeletePostByQID for the active
// deletion path, which also invalidates the render cache and issues a CDN purge.
func (s *Service) DeletePost(ctx context.Context, id int) error {
	_, err := s.postRepo.DeleteByID(ctx, id)
	return service.WrapNotFoundOrInternal(err, "post not found", "delete post failed")
}

// DeletePostByQID deletes a post by QID, scoped to ownerID when non-zero
// (JWT owner path). An ownerID of 0 deletes without an owner constraint
// (admin path). It removes the DB row, drops both render-cache variants
// synchronously, and enqueues a best-effort CDN cache-tag purge
// asynchronously. A failed purge is logged and swallowed; the CDN falls back to
// its natural TTL. Returns ErrNotFound when no row matched (wrong QID, or the
// post belongs to a different owner).
func (s *Service) DeletePostByQID(ctx context.Context, qid string, ownerID int) error {
	affected, err := s.postRepo.DeleteByQID(ctx, qid, ownerID)
	if err != nil {
		return service.WrapNotFoundOrInternal(err, "post not found", "delete post failed")
	}
	if affected == 0 {
		return service.New(service.ErrNotFound, "post not found")
	}

	s.invalidateCache(qid)

	purgeCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	go func() {
		defer cancel()
		s.purger.PurgePost(purgeCtx, qid)
	}()

	return nil
}

// invalidateCache removes both render-cache variants for a QID. Called
// synchronously on every deletion path (user delete, admin delete, prune).
func (s *Service) invalidateCache(qid string) {
	s.cache.Delete(cacheKey(qid, "html"))
	s.cache.Delete(cacheKey(qid, "raw"))
}

// PruneExpired deletes expired posts based on retention days. It removes the
// DB rows and origin render-cache entries but does NOT issue CDN purges —
// stale delivery of already-expired ephemeral content is harmless and the prune
// volume could be large.
func (s *Service) PruneExpired(ctx context.Context, retentionDays, batchSize int) error {
	if retentionDays <= 0 {
		return service.New(service.ErrValidation, "retention days must be positive")
	}
	if batchSize <= 0 {
		batchSize = 99
	}

	qids, err := s.postRepo.PruneExpired(ctx, retentionDays, batchSize)
	if err != nil {
		return service.Wrap(service.ErrInternal, "prune expired posts failed", err)
	}

	for _, qid := range qids {
		s.invalidateCache(qid)
	}

	return nil
}

// CountExpired counts expired posts based on retention days.
func (s *Service) CountExpired(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, service.New(service.ErrValidation, "retention days must be positive")
	}

	count, err := s.postRepo.CountExpired(ctx, retentionDays)
	if err != nil {
		return 0, service.Wrap(service.ErrInternal, "count expired posts failed", err)
	}

	return count, nil
}
