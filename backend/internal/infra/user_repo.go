// Package infra provides infrastructure layer implementations.
package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"markpost/internal/domain/user"
	"markpost/pkg/utils"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

// UserRepository provides user data access operations.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *gorm.DB) user.Repository {
	return &UserRepository{db: db}
}

// GetByPostKey retrieves a user by their post key.
func (r *UserRepository) GetByPostKey(ctx context.Context, postKey string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("post_key = ?", postKey).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("GetByPostKey: %w", err)
	}
	return &u, nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id int) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	return &u, nil
}

// GetByGitHubID retrieves a user by their GitHub ID.
func (r *UserRepository) GetByGitHubID(ctx context.Context, githubID int64) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("github_id = ?", githubID).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("GetByGitHubID: %w", err)
	}
	return &u, nil
}

// GetByUsername retrieves a user by their username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("GetByUsername: %w", err)
	}
	return &u, nil
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("GetByEmail: %w", err)
	}
	return &u, nil
}

// Create creates a new user with the provided credentials.
func (r *UserRepository) Create(ctx context.Context, email, username, password string) (*user.User, error) {
	exists, err := r.existsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("email is already taken")
	}

	exists, err = r.existsByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("username is already taken")
	}

	return r.createWithUniquePostKey(ctx, email, username, password, nil, nil)
}

// CreateFromGitHub creates a new user from GitHub user data.
func (r *UserRepository) CreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	email := githubUser.Email
	if email == "" {
		email = fmt.Sprintf("%d@github.local", githubUser.ID)
	}
	return r.createWithUniquePostKey(ctx, email, githubUser.Login, "", &githubUser.ID, &githubUser.AvatarURL)
}

// GetOrCreateFromGitHub retrieves or creates a user from GitHub user data.
func (r *UserRepository) GetOrCreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	u, err := r.GetByGitHubID(ctx, githubUser.ID)
	if err == nil {
		return u, nil
	}

	if !errors.Is(err, user.ErrNotFound) {
		return nil, err
	}

	return r.CreateFromGitHub(ctx, githubUser)
}

// ValidatePassword validates a user's password and returns the user if valid.
func (r *UserRepository) ValidatePassword(ctx context.Context, email, password string) (*user.User, error) {
	u, err := r.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if u.Password == "" {
		return nil, fmt.Errorf("user has no password set")
	}

	ok, err := utils.CheckPassword(password, u.Password)
	if err != nil {
		return nil, fmt.Errorf("validate user %s password: %w", email, err)
	}
	if !ok {
		return nil, fmt.Errorf("invalid password")
	}

	return u, nil
}

// SetPassword updates a user's password.
func (r *UserRepository) SetPassword(ctx context.Context, userID int, password string) error {
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("password_hash", hashed).Error
}

// SetRole updates a user's role.
func (r *UserRepository) SetRole(ctx context.Context, userID int, role user.Role) error {
	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("role", role).Error
}

// DeleteByID deletes a user by their ID.
func (r *UserRepository) DeleteByID(ctx context.Context, userID int) (int64, error) {
	tx := r.db.WithContext(ctx).Delete(&user.User{}, userID)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

// GetAll retrieves all users with pagination.
func (r *UserRepository) GetAll(ctx context.Context, offset, limit int) ([]user.User, error) {
	var users []user.User
	err := r.db.WithContext(ctx).Order("id asc").Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("GetAll: %w", err)
	}
	return users, nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&user.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("Count: %w", err)
	}
	return count, nil
}

// UpdateLastLoginAt updates the last login timestamp for a user.
func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, userID int, lastLoginAt time.Time) error {
	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("last_login_at", lastLoginAt).Error
}

func (r *UserRepository) existsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&user.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("existsByEmail: %w", err)
	}
	return count > 0, nil
}

func (r *UserRepository) existsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&user.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("existsByUsername: %w", err)
	}
	return count > 0, nil
}

func (r *UserRepository) existsByPostKey(ctx context.Context, postKey string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&user.User{}).Where("post_key = ?", postKey).Count(&count).Error; err != nil {
		return false, fmt.Errorf("existsByPostKey: %w", err)
	}
	return count > 0, nil
}

func (r *UserRepository) createWithUniquePostKey(ctx context.Context, email, username, password string, githubID *int64, avatarURL *string) (*user.User, error) {
	for {
		u, err := r.makeUser(email, username, password, githubID, avatarURL)
		if err != nil {
			return nil, err
		}

		exists, err := r.existsByPostKey(ctx, u.PostKey)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		if err = r.db.WithContext(ctx).Create(u).Error; err == nil {
			return u, nil
		}

		return nil, err
	}
}

func (r *UserRepository) makeUser(email, username, password string, githubID *int64, avatarURL *string) (*user.User, error) {
	postKey, err := gonanoid.New()
	if err != nil {
		return nil, err
	}

	var hash string
	if password != "" {
		hash, err = utils.HashPassword(password)
		if err != nil {
			return nil, err
		}
	}

	u := user.User{
		Email:     email,
		Username:  username,
		Password:  hash,
		PostKey:   postKey,
		GitHubID:  githubID,
		AvatarURL: avatarURL,
		IsActive:  true,
	}

	return &u, nil
}
