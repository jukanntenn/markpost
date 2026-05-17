// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"fmt"
	"time"

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
	p, err := findFirst[post.Post](ctx, r.db.Where("qid = ?", qid), post.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("GetByQID: %w", err)
	}
	return p, nil
}

// GetByID retrieves a post by its ID.
func (r *PostRepository) GetByID(ctx context.Context, id int) (*post.Post, error) {
	p, err := findFirst[post.Post](ctx, r.db.Where("id = ?", id), post.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	return p, nil
}

// CountByUserID counts posts for a specific user.
func (r *PostRepository) CountByUserID(ctx context.Context, userID int) (int64, error) {
	return countQuery(ctx, r.db.Model(&post.Post{}).Where("user_id = ?", userID), "CountByUserID")
}

// GetByUserID retrieves posts for a specific user with pagination.
func (r *PostRepository) GetByUserID(ctx context.Context, userID int, offset int, limit int) ([]post.Post, error) {
	return findMany[post.Post](ctx, r.db.Where("user_id = ?", userID).Order("created_at DESC"), offset, limit, "GetByUserID")
}

// ListAll retrieves all posts with optional search and pagination.
func (r *PostRepository) ListAll(ctx context.Context, search string, offset int, limit int) ([]post.Post, error) {
	query := r.applySearch(r.db.Model(&post.Post{}), search).Preload("User").Order("created_at DESC")
	return findMany[post.Post](ctx, query, offset, limit, "ListAll")
}

// CountAll counts all posts with optional search filter.
func (r *PostRepository) CountAll(ctx context.Context, search string) (int64, error) {
	query := r.applySearch(r.db.Model(&post.Post{}), search)
	return countQuery(ctx, query, "CountAll")
}

// UpdateByID updates a post by its ID.
func (r *PostRepository) UpdateByID(ctx context.Context, id int, title string, body string) (*post.Post, error) {
	updates := map[string]any{
		"title": title,
		"body":  body,
	}
	result := r.db.WithContext(ctx).Model(&post.Post{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("UpdateByID: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("UpdateByID: %w", post.ErrNotFound)
	}

	p, err := findFirst[post.Post](ctx, r.db.Preload("User").Where("id = ?", id), post.ErrNotFound)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// DeleteByID deletes a post by its ID.
func (r *PostRepository) DeleteByID(ctx context.Context, id int) (int64, error) {
	n, err := deleteWhere[post.Post](ctx, r.db.Where("id = ?", id))
	if err != nil {
		return 0, fmt.Errorf("DeleteByID: %w", err)
	}
	return n, nil
}

// PruneExpired deletes expired posts based on retention days.
func (r *PostRepository) PruneExpired(ctx context.Context, retentionDays int, batchSize int) error {
	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)

	for {
		ids, err := r.getIDsBefore(ctx, expiredBefore, batchSize)
		if err != nil {
			return fmt.Errorf("PruneExpired: %w", err)
		}

		if len(ids) == 0 {
			break
		}

		deleted, err := r.deleteByIDs(ctx, ids)
		if err != nil {
			return fmt.Errorf("PruneExpired: %w", err)
		}

		if deleted < int64(batchSize) {
			break
		}
	}

	return nil
}

// CountExpired counts expired posts based on retention days.
func (r *PostRepository) CountExpired(ctx context.Context, retentionDays int) (int64, error) {
	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
	return countQuery(ctx, r.db.Model(&post.Post{}).Where("created_at < ?", expiredBefore), "CountExpired")
}

func (r *PostRepository) getIDsBefore(ctx context.Context, before time.Time, limit int) ([]int, error) {
	var ids []int

	queryBuilder := r.db.WithContext(ctx).Model(&post.Post{}).Where("created_at < ?", before)
	if limit > 0 {
		queryBuilder = queryBuilder.Limit(limit)
	}

	if err := queryBuilder.Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("getIDsBefore: %w", err)
	}

	return ids, nil
}

func (r *PostRepository) deleteByIDs(ctx context.Context, ids []int) (int64, error) {
	tx := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&post.Post{})
	if tx.Error != nil {
		return 0, fmt.Errorf("deleteByIDs: %w", tx.Error)
	}

	return tx.RowsAffected, nil
}

func (r *PostRepository) applySearch(query *gorm.DB, search string) *gorm.DB {
	if search != "" {
		return query.Where("title LIKE ?", likeContains(search))
	}
	return query
}
