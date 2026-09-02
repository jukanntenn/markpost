package infra

import (
	"context"
	"fmt"
	"time"

	"markpost/internal/domain"
	"markpost/internal/domain/post"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

// PostRepository provides post data access operations.
type PostRepository struct {
	db *gorm.DB
}

// NewPostRepository creates a new PostRepository instance.
func NewPostRepository(db *gorm.DB) post.Repository {
	return &PostRepository{db: db}
}

// Create creates a new post.
func (r *PostRepository) Create(ctx context.Context, title, body string, userID int) (*post.Post, error) {
	qid, err := gonanoid.New()
	if err != nil {
		return nil, err
	}

	p := post.Post{
		QID:    "p-" + qid,
		Title:  title,
		Body:   body,
		UserID: userID,
	}
	err = r.db.WithContext(ctx).Create(&p).Error
	if err != nil {
		return nil, fmt.Errorf("Create: %w", err)
	}

	return &p, nil
}

// CreateBatch creates multiple posts in a batch.
func (r *PostRepository) CreateBatch(ctx context.Context, posts []post.Post) (int, error) {
	if len(posts) == 0 {
		return 0, nil
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&posts).Error
	})

	if err != nil {
		return 0, fmt.Errorf("CreateBatch: %w", err)
	}

	return len(posts), nil
}

// GetByQID retrieves a post by its QID.
func (r *PostRepository) GetByQID(ctx context.Context, qid string) (*post.Post, error) {
	return findFirst[post.Post](ctx, r.db.Where("qid = ?", qid), domain.ErrNotFound)
}

// GetByID retrieves a post by its ID.
func (r *PostRepository) GetByID(ctx context.Context, id int) (*post.Post, error) {
	return findFirst[post.Post](ctx, r.db.Where("id = ?", id), domain.ErrNotFound)
}

// CountByUserID counts posts for a specific user.
func (r *PostRepository) CountByUserID(ctx context.Context, userID int) (int64, error) {
	return countQuery(ctx, r.db.Model(&post.Post{}).Where("user_id = ?", userID), "CountByUserID")
}

// GetByUserID retrieves posts for a specific user with pagination. search
// filters by title/body (B3.3/F.5 user posts search).
func (r *PostRepository) GetByUserID(ctx context.Context, userID int, search string, offset int, limit int) ([]post.Post, error) {
	query := applySearch(r.db.Model(&post.Post{}).Where("user_id = ?", userID), search, "title", "body").Order("created_at DESC, id DESC")
	return findMany[post.Post](ctx, query, offset, limit, "GetByUserID")
}

// CountByUserIDSearch counts posts for a specific user matching the search.
func (r *PostRepository) CountByUserIDSearch(ctx context.Context, userID int, search string) (int64, error) {
	query := applySearch(r.db.Model(&post.Post{}).Where("user_id = ?", userID), search, "title", "body")
	return countQuery(ctx, query, "CountByUserIDSearch")
}

func (r *PostRepository) searchQuery(search string) *gorm.DB {
	return applySearch(r.db.Model(&post.Post{}), search, "title", "body")
}

// ListAll retrieves all posts with optional search and pagination. When
// username is non-empty, only posts by a user with that username are returned
// (F.9 admin 帖子用户筛选).
func (r *PostRepository) ListAll(ctx context.Context, search, username string, offset int, limit int) ([]post.Post, error) {
	query := r.searchQuery(search).Preload("User").Order("created_at DESC, id DESC")
	if username != "" {
		query = query.Joins("JOIN users ON users.id = posts.user_id").
			Where("users.username ILIKE ?", likeContains(username))
	}
	return findMany[post.Post](ctx, query, offset, limit, "ListAll")
}

// CountAll counts all posts with optional search + username filter.
func (r *PostRepository) CountAll(ctx context.Context, search, username string) (int64, error) {
	query := r.searchQuery(search)
	if username != "" {
		query = query.Joins("JOIN users ON users.id = posts.user_id").
			Where("users.username ILIKE ?", likeContains(username))
	}
	return countQuery(ctx, query, "CountAll")
}

// CountSince counts posts created at or after since (stats week delta, D2.4).
func (r *PostRepository) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return countQuery(ctx, r.db.Model(&post.Post{}).Where("created_at >= ?", since), "CountSince")
}

// DeleteByID deletes a post by its ID.
func (r *PostRepository) DeleteByID(ctx context.Context, id int) (int64, error) {
	return deleteWhere[post.Post](ctx, r.db.Where("id = ?", id))
}

// DeleteByQID deletes a post by its QID. When ownerID is non-zero the row is
// only deleted if it belongs to that owner (returns affected=0 otherwise); an
// ownerID of 0 deletes by QID with no owner constraint (admin path).
func (r *PostRepository) DeleteByQID(ctx context.Context, qid string, ownerID int) (int64, error) {
	q := r.db.WithContext(ctx).Where("qid = ?", qid)
	if ownerID > 0 {
		q = q.Where("user_id = ?", ownerID)
	}
	return deleteWhere[post.Post](ctx, q)
}

// PruneExpired deletes expired posts based on retention days. retentionDays is
// the global fallback; a user's retention_days overrides it per row (0 = keep
// forever, MRFC 2026-08-31-per-user-history-retention-policy). It returns the
// QIDs of the deleted posts so the caller can drop their origin render-cache
// entries. It does not issue CDN purges — stale delivery of already-expired
// ephemeral content is harmless, and prune volume can be large.
func (r *PostRepository) PruneExpired(ctx context.Context, retentionDays int, batchSize int) ([]string, error) {
	var pruned []string

	for {
		rows, err := r.getQIDsExpired(ctx, retentionDays, batchSize)
		if err != nil {
			return pruned, fmt.Errorf("PruneExpired: %w", err)
		}

		if len(rows) == 0 {
			break
		}

		qids := make([]string, 0, len(rows))
		ids := make([]int, 0, len(rows))
		for _, ro := range rows {
			qids = append(qids, ro.QID)
			ids = append(ids, ro.ID)
		}

		deleted, err := r.deleteByIDs(ctx, ids)
		if err != nil {
			return pruned, fmt.Errorf("PruneExpired: %w", err)
		}
		pruned = append(pruned, qids...)

		if deleted < int64(batchSize) {
			break
		}
	}

	return pruned, nil
}

// CountExpiringForUsers counts posts of the targeted users past the cutoff
// (retention impact preview). A nil cutoff (candidate = keep forever) matches
// nothing. vipOnly targets every VIP user instead of explicit ids.
func (r *PostRepository) CountExpiringForUsers(ctx context.Context, userIDs []int, vipOnly bool, cutoff *time.Time) (int64, error) {
	if cutoff == nil {
		return 0, nil
	}
	q := r.db.Model(&post.Post{}).Where("created_at < ?", *cutoff)
	if vipOnly {
		q = q.Where("user_id IN (SELECT id FROM users WHERE vip = TRUE)")
	} else {
		if len(userIDs) == 0 {
			return 0, nil
		}
		q = q.Where("user_id IN ?", userIDs)
	}
	return countQuery(ctx, q, "CountExpiringForUsers")
}

// CountExpired counts expired posts based on retention days (per-user
// retention_days overrides the global fallback, same predicate as PruneExpired).
func (r *PostRepository) CountExpired(ctx context.Context, retentionDays int) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM posts p JOIN users u ON u.id = p.user_id WHERE "+expiredPostsPredicate,
			globalCutoffArg(retentionDays)).
		Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("CountExpired: %w", err)
	}
	return count, nil
}

// expiredPostsPredicate is the per-row expiry predicate shared by PruneExpired
// and CountExpired. The cutoff is derived per row: an explicit user
// retention_days wins (0 = forever → the NULL cutoff excludes the row via the
// NULL comparison), otherwise the global fallback applies — and a global 0
// (never expire) passes a NULL cutoff too, excluding every inheriting row.
// posts.user_id is NOT NULL with ON DELETE CASCADE, so the JOIN always resolves.
const expiredPostsPredicate = `p.created_at < CASE
    WHEN u.retention_days = 0 THEN NULL
    WHEN u.retention_days IS NOT NULL THEN now() - make_interval(days => u.retention_days)
    ELSE ?
END`

// globalCutoffArg returns the fallback cutoff for inherit rows: now minus
// retentionDays, or nil (a NULL that excludes every such row) when the global
// config says never expire.
func globalCutoffArg(retentionDays int) any {
	if retentionDays == 0 {
		return nil
	}
	return time.Now().AddDate(0, 0, -retentionDays)
}

type expiredRow struct {
	ID  int    `gorm:"column:id"`
	QID string `gorm:"column:qid"`
}

func (r *PostRepository) getQIDsExpired(ctx context.Context, retentionDays, limit int) ([]expiredRow, error) {
	var rows []expiredRow

	sql := `SELECT p.id, p.qid FROM posts p JOIN users u ON u.id = p.user_id
	        WHERE ` + expiredPostsPredicate + `
	        ORDER BY p.created_at LIMIT ?`
	if err := r.db.WithContext(ctx).Raw(sql, globalCutoffArg(retentionDays), limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("getQIDsExpired: %w", err)
	}

	return rows, nil
}

func (r *PostRepository) deleteByIDs(ctx context.Context, ids []int) (int64, error) {
	tx := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&post.Post{})
	if tx.Error != nil {
		return 0, fmt.Errorf("deleteByIDs: %w", tx.Error)
	}

	return tx.RowsAffected, nil
}
