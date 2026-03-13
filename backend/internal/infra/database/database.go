package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/pkg/utils"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func New(dsn string) (*Database, error) {
	cfg := config.Get()

	var db *gorm.DB
	var err error

	switch cfg.DB.Driver {
	case "postgresql":
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("NewDatabase open postgres: %w", err)
		}
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("NewDatabase open sqlite: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DB.Driver)
	}

	if err = db.AutoMigrate(
		&user.User{},
		&user.RefreshToken{},
		&user.TokenBlacklist{},
		&post.Post{},
		&delivery.Channel{},
	); err != nil {
		return nil, fmt.Errorf("NewDatabase auto migrate: %w", err)
	}

	database := &Database{db: db}

	username := cfg.Admin.InitialUsername
	exists, err := database.userExists(username)
	if err != nil {
		return nil, err
	}
	if !exists {
		password, err := utils.HashPassword(cfg.Admin.InitialPassword)
		if err != nil {
			return nil, err
		}

		postKey, err := utils.GeneratePostKey(cfg.PostKeyLength)
		if err != nil {
			return nil, err
		}

		u := user.User{
			Email:    username + "@localhost",
			Username: username,
			Password: password,
			PostKey:  postKey,
			IsActive: true,
		}
		if err = database.createUser(&u); err != nil {
			return nil, err
		}
		log.Printf("initialized user: %s", username)
	}

	return database, nil
}

func NewTestDatabase() (*Database, error) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("NewTestDatabase open sqlite: %w", err)
	}
	if sqlDB, err2 := gdb.DB(); err2 == nil {
		_, _ = sqlDB.Exec("PRAGMA journal_mode=WAL;")
		_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON;")
	}

	if err = gdb.AutoMigrate(
		&user.User{},
		&user.RefreshToken{},
		&user.TokenBlacklist{},
		&post.Post{},
		&delivery.Channel{},
	); err != nil {
		return nil, fmt.Errorf("NewTestDatabase auto migrate: %w", err)
	}

	return &Database{db: gdb}, nil
}

func (d *Database) DB() *gorm.DB {
	return d.db
}

func (d *Database) userExists(username string) (bool, error) {
	var count int64
	if err := d.db.Model(&user.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("userExists: %w", err)
	}
	return count > 0, nil
}

func (d *Database) createUser(u *user.User) error {
	return d.db.Create(u).Error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) user.Repository {
	return &UserRepository{db: db}
}

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

func (r *UserRepository) CreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	email := githubUser.Email
	if email == "" {
		email = fmt.Sprintf("%d@github.local", githubUser.ID)
	}
	return r.createWithUniquePostKey(ctx, email, githubUser.Login, "", &githubUser.ID, &githubUser.AvatarURL)
}

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

func (r *UserRepository) SetPassword(ctx context.Context, userID int, password string) error {
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("password_hash", hashed).Error
}

func (r *UserRepository) SetRole(ctx context.Context, userID int, role user.Role) error {
	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("role", role).Error
}

func (r *UserRepository) DeleteByID(ctx context.Context, userID int) (int64, error) {
	tx := r.db.WithContext(ctx).Delete(&user.User{}, userID)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

func (r *UserRepository) GetAll(ctx context.Context, offset, limit int) ([]user.User, error) {
	var users []user.User
	err := r.db.WithContext(ctx).Order("id asc").Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("GetAll: %w", err)
	}
	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&user.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("Count: %w", err)
	}
	return count, nil
}

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

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) user.TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	token := user.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&token).Error
}

func (r *TokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	var token user.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("GetRefreshToken: %w", err)
	}
	return &token, nil
}

func (r *TokenRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&user.RefreshToken{}).Error
}

func (r *TokenRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID int) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&user.RefreshToken{}).Error
}

func (r *TokenRepository) StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	blacklist := user.TokenBlacklist{
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&blacklist).Error
}

func (r *TokenRepository) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&user.TokenBlacklist{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("IsTokenBlacklisted: %w", err)
	}
	return count > 0, nil
}

func (r *TokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&user.RefreshToken{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&user.TokenBlacklist{}).Error
}
