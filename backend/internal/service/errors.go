// Package service provides service-level error types and interfaces.
package service

import (
	"context"
	"errors"
)

// ErrCode represents an error code for service errors.
type ErrCode string

const (
	// ErrInvalidCredentials indicates invalid login credentials.
	ErrInvalidCredentials ErrCode = "invalid_credentials"
	// ErrInvalidPassword indicates an invalid password was provided.
	ErrInvalidPassword ErrCode = "invalid_password"
	// ErrUnauthorized indicates the user is not authorized.
	ErrUnauthorized ErrCode = "unauthorized"
	// ErrInternal indicates an internal server error.
	ErrInternal ErrCode = "internal"
	// ErrValidation indicates a validation error.
	ErrValidation ErrCode = "validation"
	// ErrNotFound indicates a resource was not found.
	ErrNotFound ErrCode = "not_found"

	// ErrFailedGetUser indicates a failure to retrieve user data.
	ErrFailedGetUser ErrCode = "failed_get_user"

	// ErrMissingAuthorizationHeader indicates a missing authorization header.
	ErrMissingAuthorizationHeader ErrCode = "missing_authorization_header"
	// ErrInvalidToken indicates an invalid token was provided.
	ErrInvalidToken ErrCode = "invalid_token"
	// ErrInvalidPostKey indicates an invalid post key was provided.
	ErrInvalidPostKey ErrCode = "invalid_post_key"
	// ErrUserDisabled indicates the user account is disabled.
	ErrUserDisabled ErrCode = "user_disabled"

	// ErrMissingStateParam indicates a missing state parameter.
	ErrMissingStateParam ErrCode = "missing_state_param"
	// ErrMissingCode indicates a missing authorization code.
	ErrMissingCode ErrCode = "missing_code"
	// ErrInvalidRequest indicates an invalid request.
	ErrInvalidRequest ErrCode = "invalid_request"

	// ErrRequired indicates a required field is missing.
	ErrRequired ErrCode = "required"
	// ErrMinLength indicates a value does not meet minimum length requirements.
	ErrMinLength ErrCode = "min_length"
	// ErrFieldViolation indicates a field validation violation.
	ErrFieldViolation ErrCode = "validation"
)

// Error represents a structured service error with code and description.
type Error struct {
	Code        ErrCode
	Description string
	Err         error
	Details     []Error
}

func (e *Error) Error() string {
	if e.Description != "" {
		return e.Description
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return string(e.Code)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// AsServiceError attempts to convert an error to an Error.
func AsServiceError(err error) (*Error, bool) {
	var se *Error
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}

// NewServiceError creates a new Error with the given code and description.
func NewServiceError(code ErrCode, description string) *Error {
	return &Error{
		Code:        code,
		Description: description,
	}
}

// NewServiceErrorWrap creates a new Error that wraps another error.
func NewServiceErrorWrap(code ErrCode, description string, err error) *Error {
	return &Error{
		Code:        code,
		Description: description,
		Err:         err,
	}
}

// NewServiceErrorWithDetails creates a new Error with additional error details.
func NewServiceErrorWithDetails(code ErrCode, description string, details []Error) *Error {
	return &Error{
		Code:        code,
		Description: description,
		Details:     details,
	}
}

// JWTTokenPair contains an access token and refresh token pair.
type JWTTokenPair struct {
	AccessToken  string
	RefreshToken string
}

// GitHubAuthURLGenerator generates GitHub OAuth authorization URLs.
type GitHubAuthURLGenerator interface {
	GenerateGitHubAuthURL(ctx context.Context) (string, error)
}

// AuthService provides authentication operations.
type AuthService interface {
	GitHubAuthURLGenerator
	LoginWithGitHub(ctx context.Context, code string) (interface{}, *JWTTokenPair, error)
	LoginWithPassword(ctx context.Context, username, password string) (interface{}, *JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (interface{}, *JWTTokenPair, error)
	ChangePassword(ctx context.Context, userID int, current, newPassword string) error
	QueryPostKey(ctx context.Context, userID int) (string, interface{}, error)
}

// PostService provides post management operations.
type PostService interface {
	CreatePost(ctx context.Context, title, body string, userID int) (string, error)
	RenderPostHTML(ctx context.Context, qid string) (string, string, error)
	GetPostMarkdown(ctx context.Context, qid string) (string, string, error)
	GetUserPosts(ctx context.Context, userID int, page, limit int) (interface{}, int64, error)
}
