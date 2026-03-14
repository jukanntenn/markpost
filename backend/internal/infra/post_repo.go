// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"errors"
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
		QID:    qid,
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
	var p post.Post
	err := r.db.WithContext(ctx).Where("qid = ?", qid).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, post.ErrNotFound
		}
		return nil, fmt.Errorf("GetByQID: %w", err)
	}
	return &p, nil
}

// GetByID retrieves a post by its ID.
func (r *PostRepository) GetByID(ctx context.Context, id int) (*post.Post, error) {
	var p post.Post
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, post.ErrNotFound
		}
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	return &p, nil
}

// CountByUserID counts posts for a specific user.
func (r *PostRepository) CountByUserID(ctx context.Context, userID int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&post.Post{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("CountByUserID: %w", err)
	}
	return count, nil
}

// GetByUserID retrieves posts for a specific user with pagination.
func (r *PostRepository) GetByUserID(ctx context.Context, userID int, offset int, limit int) ([]post.Post, error) {
	var posts []post.Post
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("GetByUserID: %w", err)
	}
	return posts, nil
}

// ListAll retrieves all posts with optional search and pagination.
func (r *PostRepository) ListAll(ctx context.Context, search string, offset int, limit int) ([]post.Post, error) {
	var posts []post.Post
	query := r.db.WithContext(ctx).Model(&post.Post{}).Preload("User").Order("created_at DESC").Offset(offset).Limit(limit)
	if search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

// CountAll counts all posts with optional search filter.
func (r *PostRepository) CountAll(ctx context.Context, search string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&post.Post{})
	if search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// UpdateByID updates a post by its ID.
func (r *PostRepository) UpdateByID(ctx context.Context, id int, title string, body string) (*post.Post, error) {
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"title": title,
		"body":  body,
	}
	if err := r.db.WithContext(ctx).Model(&post.Post{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	var p post.Post
	if err := r.db.WithContext(ctx).Preload("User").First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, post.ErrNotFound
		}
		return nil, err
	}

	return &p, nil
}

// DeleteByID deletes a post by its ID.
func (r *PostRepository) DeleteByID(ctx context.Context, id int) (int64, error) {
	tx := r.db.WithContext(ctx).Delete(&post.Post{}, id)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return tx.RowsAffected, nil
}

// PruneExpired deletes expired posts based on retention days.
func (r *PostRepository) PruneExpired(ctx context.Context, retentionDays int, batchSize int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	if batchSize <= 0 {
		batchSize = 99
	}

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
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
	var count int64
	err := r.db.WithContext(ctx).Model(&post.Post{}).Where("created_at < ?", expiredBefore).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("CountExpired: %w", err)
	}
	return count, nil
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
