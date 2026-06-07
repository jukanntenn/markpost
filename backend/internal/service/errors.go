// Package service provides service-level error types and interfaces.
package service

import (
	"errors"

	"markpost/internal/domain"
)

// ErrCode represents a service-level error code.
type ErrCode string

// Predefined error codes used throughout the service layer.
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
	ErrUserDisabled               ErrCode = "user_disabled"

	ErrForbidden ErrCode = "forbidden"

	ErrRateLimited    ErrCode = "rate_limited"
	ErrInvalidRequest ErrCode = "invalid_request"

	ErrRequired       ErrCode = "required"
	ErrMinLength      ErrCode = "min_length"
	ErrFieldViolation ErrCode = "field_violation"
)

var httpStatuses = map[ErrCode]int{
	ErrInvalidCredentials:         401,
	ErrInvalidPassword:            400,
	ErrNotFound:                   404,
	ErrUnauthorized:               401,
	ErrFailedGetUser:              500,
	ErrInternal:                   500,
	ErrValidation:                 400,
	ErrInvalidRequest:             400,
	ErrMissingAuthorizationHeader: 401,
	ErrInvalidToken:               401,
	ErrInvalidPostKey:             403,
	ErrForbidden:                  403,
	ErrRateLimited:                429,
	ErrUserDisabled:               403,
	ErrRequired:                   400,
	ErrMinLength:                  400,
	ErrFieldViolation:             400,
}

// String returns the string representation of the error code.
func (c ErrCode) String() string { return string(c) }

// HTTPStatus returns the HTTP status code associated with this error.
func (e *Error) HTTPStatus() int {
	if status, ok := httpStatuses[e.Code]; ok {
		return status
	}
	return 500
}

// FieldDetail describes a single field-level validation error.
type FieldDetail struct {
	Code        ErrCode
	Description string
}

// Error represents a service-level error with an error code and optional details.
type Error struct {
	Code        ErrCode
	Description string
	Err         error
	Details     []FieldDetail
}

// Error returns the error message, falling back to the wrapped error or error code.
func (e *Error) Error() string {
	if e.Description != "" {
		return e.Description
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return e.Code.String()
}

// Unwrap returns the underlying wrapped error.
func (e *Error) Unwrap() error {
	return e.Err
}

// AsServiceError attempts to cast an error to an Error.
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

// NewServiceErrorWrap creates a new Error that wraps an existing error.
func NewServiceErrorWrap(code ErrCode, description string, err error) *Error {
	return &Error{
		Code:        code,
		Description: description,
		Err:         err,
	}
}

// WrapNotFoundOrInternal wraps an error as either a not-found or internal service error.
func WrapNotFoundOrInternal(err error, notFoundMsg, internalMsg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) {
		return NewServiceErrorWrap(ErrNotFound, notFoundMsg, err)
	}
	return NewServiceErrorWrap(ErrInternal, internalMsg, err)
}

// NewBindingError creates a validation ServiceError with the given field details.
func NewBindingError(details []FieldDetail) *Error {
	return &Error{
		Code:        ErrValidation,
		Description: "request validation failed",
		Details:     details,
	}
}

// NewServiceErrorDetails creates a new Error with the given code, description, and field details.
func NewServiceErrorDetails(code ErrCode, description string, details []FieldDetail) *Error {
	return &Error{
		Code:        code,
		Description: description,
		Details:     details,
	}
}
