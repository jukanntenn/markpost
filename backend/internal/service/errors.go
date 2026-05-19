// Package service provides service-level error types and interfaces.
package service

import (
	"errors"

	"markpost/internal/domain"
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

func (c ErrCode) String() string { return string(c) }

func (e *ServiceError) HTTPStatus() int {
	if status, ok := httpStatuses[e.Code]; ok {
		return status
	}
	return 500
}

type FieldDetail struct {
	Code        ErrCode
	Description string
}

type ServiceError struct {
	Code        ErrCode
	Description string
	Err         error
	Details     []FieldDetail
}

func (e *ServiceError) Error() string {
	if e.Description != "" {
		return e.Description
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return e.Code.String()
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

func WrapNotFoundOrInternal(err error, notFoundMsg, internalMsg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) {
		return NewServiceErrorWrap(ErrNotFound, notFoundMsg, err)
	}
	return NewServiceErrorWrap(ErrInternal, internalMsg, err)
}

func NewBindingError(details []FieldDetail) *ServiceError {
	return &ServiceError{
		Code:        ErrValidation,
		Description: "request validation failed",
		Details:     details,
	}
}

func NewServiceErrorDetails(code ErrCode, description string, details []FieldDetail) *ServiceError {
	return &ServiceError{
		Code:        code,
		Description: description,
		Details:     details,
	}
}
