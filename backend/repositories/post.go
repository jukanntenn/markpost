package repositories

import (
	"fmt"
	"time"

	"markpost/models"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

type PostRepoInterface interface {
	CreatePost(title, body string, userID int) (*models.Post, error)
	CreatePosts(posts []models.Post) (int, error)
	GetPostByQID(qid string) (*models.Post, error)
	CountPostsByUserID(userID int) (int64, error)
	GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error)

	PruneExpiredPosts(retentionDays int, batchSize int) error
	CountExpiredPosts(retentionDays int) (int64, error)
}

type PostRepo struct {
	database *models.Database
}

func NewPostRepo(database *models.Database) PostRepoInterface {
	return &PostRepo{database: database}
}

func (r *PostRepo) CreatePost(title, body string, userID int) (*models.Post, error) {
	qid, err := gonanoid.New()
	if err != nil {
		return nil, err
	}

	post := models.Post{
		QID:    qid,
		Title:  title,
		Body:   body,
		UserID: userID,
	}
	err = post.Create(r.database)
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *PostRepo) CreatePosts(posts []models.Post) (int, error) {
	if len(posts) == 0 {
		return 0, nil
	}

	db := r.database.DB()

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&posts).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("CreatePosts: %w", err)
	}

	return len(posts), nil
}

func (r *PostRepo) GetPostByQID(qid string) (*models.Post, error) {
	post, err := models.GetPost(r.database, map[string]any{"qid": qid})
	if err == nil {
		return post, nil
	}

	return nil, err
}

func (r *PostRepo) CountPostsByUserID(userID int) (int64, error) {
	count, err := models.CountPosts(r.database, map[string]any{"user_id": userID})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostRepo) GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error) {
	posts, err := models.GetPosts(r.database, map[string]any{"user_id": userID}, offset, limit)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepo) PruneExpiredPosts(retentionDays int, batchSize int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	if batchSize <= 0 {
		batchSize = 99
	}

	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)

	for {
		ids, err := models.GetPostIDsBefore(r.database, expiredBefore, batchSize)
		if err != nil {
			return fmt.Errorf("PruneExpiredPosts: %w", err)
		}

		if len(ids) == 0 {
			break
		}

		deleted, err := models.DeletePostsByIDs(r.database, ids)
		if err != nil {
			return fmt.Errorf("PruneExpiredPosts: %w", err)
		}

		if deleted < int64(batchSize) {
			break
		}
	}

	return nil
}

func (r *PostRepo) CountExpiredPosts(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
	return models.CountPostsBefore(r.database, expiredBefore)
}
