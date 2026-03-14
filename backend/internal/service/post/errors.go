// Package post provides post-related business logic and services.
package post

import "errors"

// ErrCode represents an error code type.
type ErrCode string

const (
	// ErrNotFound indicates a resource was not found.
	ErrNotFound ErrCode = "not_found"
	// ErrInternal indicates an internal server error.
	ErrInternal ErrCode = "internal"
	// ErrValidation indicates a validation error.
	ErrValidation ErrCode = "validation"
)

// ServiceError represents a service-level error.
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

// AsServiceError attempts to convert an error to ServiceError.
func AsServiceError(err error) (*ServiceError, bool) {
	var se *ServiceError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}

// NewServiceError creates a new ServiceError.
func NewServiceError(code ErrCode, description string) *ServiceError {
	return &ServiceError{
		Code:        code,
		Description: description,
	}
}

// NewServiceErrorWrap creates a new ServiceError with wrapped error.
func NewServiceErrorWrap(code ErrCode, description string, err error) *ServiceError {
	return &ServiceError{
		Code:        code,
		Description: description,
		Err:         err,
	}
}
