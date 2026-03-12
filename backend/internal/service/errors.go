package service

import (
	"context"
	"errors"
)

type ErrCode string

const (
	ErrInvalidCredentials ErrCode = "invalid_credentials"
	ErrInvalidPassword    ErrCode = "invalid_password"
	ErrUnauthorized       ErrCode = "unauthorized"
	ErrInternal           ErrCode = "internal"
	ErrValidation         ErrCode = "validation"
	ErrNotFound           ErrCode = "not_found"

	ErrFailedGetUser ErrCode = "failed_get_user"

	ErrMissingAuthorizationHeader ErrCode = "missing_authorization_header"
	ErrInvalidToken               ErrCode = "invalid_token"
	ErrInvalidPostKey             ErrCode = "invalid_post_key"

	ErrMissingStateParam ErrCode = "missing_state_param"
	ErrMissingCode       ErrCode = "missing_code"
	ErrInvalidRequest    ErrCode = "invalid_request"

	ErrRequired       ErrCode = "required"
	ErrMinLength      ErrCode = "min_length"
	ErrFieldViolation ErrCode = "validation"
)

type ServiceError struct {
	Code        ErrCode
	Description string
	Err         error
	Details     []ServiceError
}

func (e *ServiceError) Error() string {
	if e.Description != "" {
		return e.Description
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return string(e.Code)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

func AsServiceError(err error) (*ServiceError, bool) {
	var se *ServiceError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}

func NewServiceError(code ErrCode, description string) *ServiceError {
	return &ServiceError{
		Code:        code,
		Description: description,
	}
}

func NewServiceErrorWrap(code ErrCode, description string, err error) *ServiceError {
	return &ServiceError{
		Code:        code,
		Description: description,
		Err:         err,
	}
}

func NewServiceErrorWithDetails(code ErrCode, description string, details []ServiceError) *ServiceError {
	return &ServiceError{
		Code:        code,
		Description: description,
		Details:     details,
	}
}

type JWTTokenPair struct {
	AccessToken  string
	RefreshToken string
}

type GitHubAuthURLGenerator interface {
	GenerateGitHubAuthURL(ctx context.Context) (string, error)
}

type AuthService interface {
	GitHubAuthURLGenerator
	LoginWithGitHub(ctx context.Context, code string) (interface{}, *JWTTokenPair, error)
	LoginWithPassword(ctx context.Context, username, password string) (interface{}, *JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (interface{}, *JWTTokenPair, error)
	ChangePassword(ctx context.Context, userID int, current, new string) error
	QueryPostKey(ctx context.Context, userID int) (string, interface{}, error)
}

type PostService interface {
	CreatePost(ctx context.Context, title, body string, userID int) (string, error)
	RenderPostHTML(ctx context.Context, qid string) (string, string, error)
	GetPostMarkdown(ctx context.Context, qid string) (string, string, error)
	GetUserPosts(ctx context.Context, userID int, page, limit int) (interface{}, int64, error)
}
