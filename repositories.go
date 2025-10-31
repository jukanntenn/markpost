package main

import (
	"database/sql"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserByPostKey(postKey string) (*User, error)
	GetUserByID(id int) (*User, error)
	GetUserByGitHubID(githubID int64) (*User, error)
	GetUserByUsername(username string) (*User, error)
	CreateUserFromGitHubUser(githubUser *GitHubUser) (*User, error)
	GetOrCreateUserFromGitHubUser(githubUser *GitHubUser) (*User, error)
	CreateUserWithPassword(username, password string) (*User, error)
	ValidateUserPassword(username, password string) (*User, error)
	UpdatePassword(userID int, hashed string) error
}

type PostRepository interface {
	CreatePost(title, body string, userID ...int) (*Post, error)
	CreatePostWithUser(title, body string, userID int) (*Post, error)
	GetPostByID(id string) (*Post, error)
	GetPostsByUserID(userID int) ([]Post, error)
	GetPostsByUserIDPaginated(userID int, page int, limit int) ([]Post, int64, error)
	CleanupExpiredPosts(retentionDays int, batchSize int) error
	GetExpiredPostsCount(retentionDays int) (int64, error)
	PreviewExpiredPosts(retentionDays int, limit int) ([]Post, error)
}

type userRepository struct {
	db *gorm.DB
}

type postRepository struct {
	db *gorm.DB
}

func makeUser(username, password string, githubID *int64) (*User, error) {
	postKey, err := GeneratePostKey(8)
	if err != nil {
		return nil, err
	}

	var hashed string
	if password != "" {
		hashed, err = HashPassword(password)
		if err != nil {
			return nil, err
		}
	}

	user := User{
		Username: username,
		Password: hashed,
		PostKey:  postKey,
		GitHubID: githubID,
	}

	return &user, nil
}

func (r *userRepository) GetUserByPostKey(postKey string) (*User, error) {
	var user User
	err := r.db.Take(&user, "post_key = ?", postKey).Error
	if err == nil {
		return &user, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, sql.ErrNoRows
	}

	return nil, err
}

func (r *userRepository) GetUserByID(id int) (*User, error) {
	var user User
	err := r.db.Take(&user, "id = ?", id).Error
	if err == nil {
		return &user, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, sql.ErrNoRows
	}

	return nil, err
}

func (r *userRepository) GetUserByGitHubID(githubID int64) (*User, error) {
	var user User
	err := r.db.Take(&user, "github_id = ?", githubID).Error
	if err == nil {
		return &user, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, sql.ErrNoRows
	}

	return nil, err
}

func (r *userRepository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.db.Take(&user, "username = ?", username).Error
	if err == nil {
		return &user, nil
	}

	if err == gorm.ErrRecordNotFound {
		return nil, sql.ErrNoRows
	}

	return nil, err
}

func (r *userRepository) CreateUserFromGitHubUser(githubUser *GitHubUser) (*User, error) {
	user, err := makeUser(githubUser.Login, "", &githubUser.ID)
	if err != nil {
		return nil, err
	}

	err = r.db.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepository) GetOrCreateUserFromGitHubUser(githubUser *GitHubUser) (*User, error) {
	user, err := r.GetUserByGitHubID(githubUser.ID)
	if err == nil {
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	return r.CreateUserFromGitHubUser(githubUser)
}

func (r *userRepository) CreateUserWithPassword(username, password string) (*User, error) {
	var count int64
	if err := r.db.Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, fmt.Errorf("username is already taken")
	}

	user, err := makeUser(username, password, nil)
	if err != nil {
		return nil, err
	}

	err = r.db.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepository) ValidateUserPassword(username, password string) (*User, error) {
	user, err := r.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	if user.Password == "" {
		return nil, fmt.Errorf("user has no password set")
	}

	if err := CheckPassword(password, user.Password); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

func (r *userRepository) UpdatePassword(userID int, hashed string) error {
	return r.db.Model(&User{}).Where("id = ?", userID).Update("password", hashed).Error
}

func (r *postRepository) CreatePost(title, body string, userID int) (*Post, error) {
	qid, err := gonanoid.New()
	if err != nil {
		return nil, err
	}

	post := Post{
		QID:    qid,
		Title:  title,
		Body:   body,
		UserID: userID,
	}
	err = r.db.Create(&post).Error
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *postRepository) CreatePostWithUser(title, body string, userID int) (*Post, error) {
	return r.CreatePost(title, body, userID)
}

func (r *postRepository) GetPostByID(id string) (*Post, error) {
	var post Post
	err := r.db.Where("id = ?", id).First(&post).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &post, nil
}

func (r *postRepository) GetPostsByUserID(userID int) ([]Post, error) {
	var posts []Post
	err := r.db.Where("user_id = ?", userID).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *postRepository) GetPostsByUserIDPaginated(userID int, page int, limit int) ([]Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	var total int64
	if err := r.db.Model(&Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var posts []Post
	offset := (page - 1) * limit
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&posts).Error
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *postRepository) CleanupExpiredPosts(retentionDays int, batchSize int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
	for {
		var postIDs []string
		result := r.db.Model(&Post{}).
			Select("id").
			Where("created_at < ?", expiredBefore).
			Limit(batchSize).
			Pluck("id", &postIDs)
		if result.Error != nil {
			return fmt.Errorf("failed to query expired posts: %v", result.Error)
		}
		if len(postIDs) == 0 {
			break
		}
		result = r.db.Where("id IN ?", postIDs).Delete(&Post{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete post batch: %v", result.Error)
		}
		if result.RowsAffected < int64(batchSize) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (r *postRepository) GetExpiredPostsCount(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}
	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
	var count int64
	result := r.db.Model(&Post{}).Where("created_at < ?", expiredBefore).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count expired posts: %v", result.Error)
	}
	return count, nil
}

func (r *postRepository) PreviewExpiredPosts(retentionDays int, limit int) ([]Post, error) {
	if retentionDays <= 0 {
		return nil, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}
	if limit <= 0 {
		limit = 10
	}
	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
	var posts []Post
	result := r.db.Where("created_at < ?", expiredBefore).
		Order("created_at ASC").
		Limit(limit).
		Find(&posts)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to preview expired posts: %v", result.Error)
	}
	return posts, nil
}
