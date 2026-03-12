package user

import "context"

type Repository interface {
	GetByPostKey(ctx context.Context, postKey string) (*User, error)
	GetByID(ctx context.Context, id int) (*User, error)
	GetByGitHubID(ctx context.Context, githubID int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Create(ctx context.Context, username, password string) (*User, error)
	CreateFromGitHub(ctx context.Context, githubUser *GitHubUser) (*User, error)
	GetOrCreateFromGitHub(ctx context.Context, githubUser *GitHubUser) (*User, error)
	ValidatePassword(ctx context.Context, username, password string) (*User, error)
	SetPassword(ctx context.Context, userID int, password string) error
	SetRole(ctx context.Context, userID int, role Role) error
	DeleteByID(ctx context.Context, userID int) (int64, error)
	GetAll(ctx context.Context, offset, limit int) ([]User, error)
	Count(ctx context.Context) (int64, error)
}
