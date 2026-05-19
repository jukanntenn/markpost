package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"markpost/internal/domain"
	"markpost/internal/domain/user"
	"markpost/pkg/utils"

	"gorm.io/gorm"
)

// UserRepository provides user data access operations.
type UserRepository struct {
	db            *gorm.DB
	postKeyLength int
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *gorm.DB, postKeyLength int) user.Repository {
	return &UserRepository{db: db, postKeyLength: postKeyLength}
}

func (r *UserRepository) ensureEmailUnique(ctx context.Context, email string) error {
	exists, err := r.existsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%s: %w", "Create", domain.ErrEmailTaken)
	}
	return nil
}

func (r *UserRepository) findBy(ctx context.Context, name, field string, value any) (*user.User, error) {
	u, err := findFirst[user.User](ctx, r.db.Where(field+" = ?", value), domain.ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return u, nil
}

// GetByPostKey retrieves a user by their post key.
func (r *UserRepository) GetByPostKey(ctx context.Context, postKey string) (*user.User, error) {
	return r.findBy(ctx, "GetByPostKey", "post_key", postKey)
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id int) (*user.User, error) {
	return r.findBy(ctx, "GetByID", "id", id)
}

// GetByGitHubID retrieves a user by their GitHub ID.
func (r *UserRepository) GetByGitHubID(ctx context.Context, githubID int64) (*user.User, error) {
	return r.findBy(ctx, "GetByGitHubID", "github_id", githubID)
}

// GetByUsername retrieves a user by their username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	return r.findBy(ctx, "GetByUsername", "username", username)
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.findBy(ctx, "GetByEmail", "email", email)
}

// Create creates a new user with the provided credentials.
func (r *UserRepository) Create(ctx context.Context, email, username, password string) (*user.User, error) {
	if err := r.ensureEmailUnique(ctx, email); err != nil {
		return nil, err
	}

	exists, err := r.existsByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%s: %w", "Create", domain.ErrUsernameTaken)
	}

	return r.createWithUniquePostKey(ctx, email, username, password, nil, nil, r.postKeyLength)
}

// CreateFromGitHub creates a new user from GitHub user data.
func (r *UserRepository) CreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	email := githubUser.Email
	if email == "" {
		email = fmt.Sprintf("%d@github.local", githubUser.ID)
	}

	if err := r.ensureEmailUnique(ctx, email); err != nil {
		return nil, err
	}

	return r.createWithUniquePostKey(ctx, email, githubUser.Login, "", &githubUser.ID, &githubUser.AvatarURL, r.postKeyLength)
}

// GetOrCreateFromGitHub retrieves or creates a user from GitHub user data.
func (r *UserRepository) GetOrCreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	u, err := r.GetByGitHubID(ctx, githubUser.ID)
	if err == nil {
		return u, nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	return r.CreateFromGitHub(ctx, githubUser)
}

// ValidatePassword validates a user's password and returns the user if valid.
func (r *UserRepository) ValidatePassword(ctx context.Context, username, password string) (*user.User, error) {
	u, err := r.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if u.Password == "" {
		return nil, fmt.Errorf("%s: %w", "ValidatePassword", domain.ErrNoPassword)
	}

	ok, err := utils.CheckPassword(password, u.Password)
	if err != nil {
		return nil, fmt.Errorf("validate user %s password: %w", username, err)
	}
	if !ok {
		return nil, fmt.Errorf("%s: %w", "ValidatePassword", domain.ErrBadPassword)
	}

	return u, nil
}

// SetPassword updates a user's password.
func (r *UserRepository) SetPassword(ctx context.Context, userID int, password string) error {
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	return updateByID[user.User](ctx, r.db, userID, map[string]any{"password_hash": hashed}, "SetPassword")
}

// SetRole updates a user's role.
func (r *UserRepository) SetRole(ctx context.Context, userID int, role user.Role) error {
	return updateByID[user.User](ctx, r.db, userID, map[string]any{"role": role}, "SetRole")
}

// DeleteByID deletes a user by their ID.
func (r *UserRepository) DeleteByID(ctx context.Context, userID int) (int64, error) {
	return deleteWhere[user.User](ctx, r.db.Where("id = ?", userID))
}

// GetAll retrieves all users with pagination.
func (r *UserRepository) GetAll(ctx context.Context, offset, limit int) ([]user.User, error) {
	return findMany[user.User](ctx, r.db.Order("id asc"), offset, limit, "GetAll")
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return countQuery(ctx, r.db.Model(&user.User{}), "Count")
}

// UpdateLastLoginAt updates the last login timestamp for a user.
func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, userID int, lastLoginAt time.Time) error {
	return updateByID[user.User](ctx, r.db, userID, map[string]any{"last_login_at": lastLoginAt}, "UpdateLastLoginAt")
}

func (r *UserRepository) existsByEmail(ctx context.Context, email string) (bool, error) {
	return existsBy[user.User](ctx, r.db, "email", email, "existsByEmail")
}

func (r *UserRepository) existsByUsername(ctx context.Context, username string) (bool, error) {
	return existsBy[user.User](ctx, r.db, "username", username, "existsByUsername")
}

func (r *UserRepository) existsByPostKey(ctx context.Context, postKey string) (bool, error) {
	return existsBy[user.User](ctx, r.db, "post_key", postKey, "existsByPostKey")
}

func (r *UserRepository) createWithUniquePostKey(ctx context.Context, email, username, password string, githubID *int64, avatarURL *string, postKeyLength int) (*user.User, error) {
	const maxRetries = 10

	for range maxRetries {
		u, err := makeUser(email, username, password, githubID, avatarURL, postKeyLength)
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
	return nil, fmt.Errorf("createWithUniquePostKey: failed to generate unique post key after %d attempts", maxRetries)
}

func makeUser(email, username, password string, githubID *int64, avatarURL *string, postKeyLength int) (*user.User, error) {
	postKey, err := utils.GeneratePostKey(postKeyLength)
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
