package repositories

import (
	"errors"
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
	GetPostByID(id int) (*models.Post, error)
	CountPostsByUserID(userID int) (int64, error)
	GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error)
	ListAllPosts(search string, offset int, limit int) ([]models.Post, error)
	CountAllPosts(search string) (int64, error)
	UpdatePostByID(id int, title string, body string) (*models.Post, error)
	DeletePostByID(id int) (int64, error)

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

func (r *PostRepo) GetPostByID(id int) (*models.Post, error) {
	post, err := models.GetPost(r.database, map[string]any{"id": id})
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

func (r *PostRepo) ListAllPosts(search string, offset int, limit int) ([]models.Post, error) {
	db := r.database.DB()

	var posts []models.Post
	query := db.Model(&models.Post{}).Preload("User").Order("created_at DESC").Offset(offset).Limit(limit)
	if search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepo) CountAllPosts(search string) (int64, error) {
	db := r.database.DB()

	query := db.Model(&models.Post{})
	if search != "" {
		query = query.Where("title LIKE ?", "%"+search+"%")
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostRepo) UpdatePostByID(id int, title string, body string) (*models.Post, error) {
	db := r.database.DB()

	_, err := r.GetPostByID(id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{
		"title": title,
		"body":  body,
	}
	if err := db.Model(&models.Post{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	var post models.Post
	if err := db.Preload("User").Take(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrNotFound
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostRepo) DeletePostByID(id int) (int64, error) {
	db := r.database.DB()

	tx := db.Delete(&models.Post{}, id)
	if tx.Error != nil {
		return 0, tx.Error
	}

	return tx.RowsAffected, nil
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
