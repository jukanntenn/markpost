package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	QID       string    `json:"qid" gorm:"unique;not null;column:qid"`
	Title     string    `json:"title" gorm:"not null"`
	Body      string    `json:"body" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	UserID    int       `json:"user_id" gorm:"index;not null;column:user_id"`
	User      User      `json:"user" gorm:"constraint:OnDelete:CASCADE"`
}

func (model *Post) Create(database *Database) error {
	db := database.DB()
	if err := db.Create(model).Error; err != nil {
		return fmt.Errorf("Post.Create: %w", err)
	}

	return nil
}

func GetPost(database *Database, query map[string]any) (*Post, error) {
	db := database.DB()

	var post Post
	err := db.Take(&post, query).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPost: %w", err)
	}

	return &post, nil
}

func GetPosts(database *Database, query map[string]any, offset, limit int) ([]Post, error) {
	db := database.DB()

	var models []Post
	err := db.Where(query).Offset(offset).Limit(limit).Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("GetPosts: %w", err)
	}

	return models, nil
}

func CountPosts(database *Database, query map[string]any) (int64, error) {
	db := database.DB()

	var count int64
	err := db.Model(&Post{}).Where(query).Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("CountPosts: %w", err)
	}

	return count, nil
}

func CountPostsBefore(database *Database, before time.Time) (int64, error) {
	db := database.DB()

	var count int64
	err := db.Model(&Post{}).Where("created_at < ?", before).Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("CountPostsBefore: %w", err)
	}

	return count, nil
}

func GetPostIDs(database *Database, query map[string]any, limit int) ([]int, error) {
	db := database.DB()

	var ids []int

	queryBuilder := db.Model(&Post{}).Where(query)
	if limit > 0 {
		queryBuilder = queryBuilder.Limit(limit)
	}

	if err := queryBuilder.Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("GetPostIDs: %w", err)
	}

	return ids, nil
}

func GetPostIDsBefore(database *Database, before time.Time, limit int) ([]int, error) {
	db := database.DB()

	var ids []int

	queryBuilder := db.Model(&Post{}).Where("created_at < ?", before)
	if limit > 0 {
		queryBuilder = queryBuilder.Limit(limit)
	}

	if err := queryBuilder.Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("GetPostIDsBefore: %w", err)
	}

	return ids, nil
}

func DeletePosts(database *Database, query map[string]any) (int64, error) {
	db := database.DB()

	tx := db.Where(query).Delete(&Post{})
	if tx.Error != nil {
		return 0, fmt.Errorf("DeletePosts: %w", tx.Error)
	}

	return tx.RowsAffected, nil
}

func DeletePostsByIDs(database *Database, ids []int) (int64, error) {
	db := database.DB()

	tx := db.Where("id IN ?", ids).Delete(&Post{})
	if tx.Error != nil {
		return 0, fmt.Errorf("DeletePostsByIDs: %w", tx.Error)
	}

	return tx.RowsAffected, nil
}
