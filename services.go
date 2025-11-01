package main

import (
	"bytes"
	"context"
	"database/sql"
	"time"

	"github.com/yuin/goldmark"
	"golang.org/x/oauth2"
)

type ServiceError struct {
	Code    string
	Message string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func NewServiceError(code, message string, err error) *ServiceError {
	return &ServiceError{Code: code, Message: message, Err: err}
}

const (
	ErrInvalidCredentials     = "invalid_credentials"
	ErrInvalidCurrentPassword = "invalid_current_password"
	ErrSamePassword           = "same_password"
	ErrNotFound               = "not_found"
	ErrConflict               = "conflict"
	ErrUnauthorized           = "unauthorized"
	ErrInternal               = "internal"
	ErrConversionFailed       = "conversion_failed"
	ErrValidation             = "validation"
)

type AuthService struct {
	users UserRepository
	oauth *oauth2.Config
}

func NewAuthService(users UserRepository, oauth *oauth2.Config) *AuthService {
	return &AuthService{users: users, oauth: oauth}
}

func (s *AuthService) GenerateGitHubAuthURL(ctx context.Context) (string, error) {
	state, err := GenerateState()
	if err != nil {
		return "", NewServiceError(ErrInternal, "failed to generate state", err)
	}
	return s.oauth.AuthCodeURL(state), nil
}

func (s *AuthService) LoginWithGitHub(ctx context.Context, code string) (*User, *JWTTokenPair, error) {
	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, NewServiceError(ErrUnauthorized, "oauth exchange failed", err)
	}
	githubUser, err := getGitHubUser(token)
	if err != nil {
		return nil, nil, NewServiceError(ErrUnauthorized, "oauth unauthorized", err)
	}
	user, err := s.users.GetOrCreateUserFromGitHub(&GitHubUser{ID: githubUser.ID, Login: githubUser.Login})
	if err != nil {
		return nil, nil, NewServiceError(ErrInternal, "create user failed", err)
	}
	pair, err := generateJWTTokenPair(user)
	if err != nil {
		return nil, nil, NewServiceError(ErrInternal, "generate access/refresh token pair failed", err)
	}
	return user, pair, nil
}

func (s *AuthService) LoginWithPassword(ctx context.Context, username, password string) (*User, *JWTTokenPair, error) {
	user, err := s.users.ValidateUserPassword(username, password)
	if err != nil {
		return nil, nil, NewServiceError(ErrInvalidCredentials, "invalid username or password", err)
	}
	pair, err := generateJWTTokenPair(user)
	if err != nil {
		return nil, nil, NewServiceError(ErrInternal, "generate access/refresh token pair failed", err)
	}
	return user, pair, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*User, *JWTTokenPair, error) {
	claims, err := validateJWTToken(refreshToken)
	if err != nil {
		return nil, nil, NewServiceError(ErrUnauthorized, "invalid refresh token", err)
	}

	user, err := s.users.GetUserByID(claims.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, NewServiceError(ErrUnauthorized, "user not found", err)
		}
		return nil, nil, NewServiceError(ErrInternal, "query user failed", err)
	}

	pair, err := generateJWTTokenPair(user)
	if err != nil {
		return nil, nil, NewServiceError(ErrInternal, "generate access/refresh token pair failed", err)
	}
	return user, pair, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, current, new string) error {
	user, err := s.users.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrConflict, "user not found", err)
		}
		return NewServiceError(ErrInternal, "query user failed", err)
	}
	if err = CheckPassword(current, user.Password); err != nil {
		return NewServiceError(ErrInvalidCurrentPassword, "invalid current password", err)
	}
	if current == new {
		return NewServiceError(ErrSamePassword, "new password same as current", nil)
	}
	if err := s.users.SetUserPassword(userID, new); err != nil {
		return NewServiceError(ErrInternal, "set password failed", err)
	}
	return nil
}

func (s *AuthService) QueryPostKey(ctx context.Context, userID int) (string, time.Time, error) {
	user, err := s.users.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, NewServiceError(ErrConflict, "user not found", err)
		}
		return "", time.Time{}, NewServiceError(ErrInternal, "query user failed", err)
	}
	return user.PostKey, user.CreatedAt, nil
}

type PostService struct {
	posts PostRepository
}

func NewPostService(posts PostRepository) *PostService {
	return &PostService{posts: posts}
}

func (s *PostService) CreatePost(ctx context.Context, title, body string, userID int) (string, error) {
	post, err := s.posts.CreatePostWithUser(title, body, userID)
	if err != nil {
		return "", NewServiceError(ErrInternal, "create post failed", err)
	}
	return post.QID, nil
}

func (s *PostService) RenderPostHTML(ctx context.Context, qid string) (string, string, error) {
	post, err := s.posts.GetPostByQID(qid)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", NewServiceError(ErrNotFound, "post not found", err)
		}
		return "", "", NewServiceError(ErrInternal, "query post failed", err)
	}
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(post.Body), &buf); err != nil {
		return "", "", NewServiceError(ErrConversionFailed, "markdown conversion failed", err)
	}
	return post.Title, buf.String(), nil
}

func (s *PostService) GetUserPostsPaginated(ctx context.Context, userID int, page int, limit int) ([]Post, int64, error) {
	posts, total, err := s.posts.GetPostsByUserIDPaginated(userID, page, limit)
	if err != nil {
		return nil, 0, NewServiceError(ErrInternal, "query posts failed", err)
	}
	return posts, total, nil
}
