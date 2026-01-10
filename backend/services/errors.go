package services

import "errors"

type ErrCode string

const (
	ErrInvalidCredentials ErrCode = "invalid_credentials"
	ErrInvalidPassword    ErrCode = "invalid_password"
	ErrUnauthorized       ErrCode = "unauthorized"
	ErrInternal           ErrCode = "internal"
	ErrValidation         ErrCode = "validation"
	ErrNotFound           ErrCode = "not_found"

	ErrFailedGetUser ErrCode = "failed_get_user"

	// http-layer specific errors
	ErrMissingAuthorizationHeader ErrCode = "missing_authorization_header"
	ErrInvalidToken               ErrCode = "invalid_token"
	ErrInvalidPostKey             ErrCode = "invalid_post_key"

	ErrMissingStateParam ErrCode = "missing_state_param"
	ErrMissingCode       ErrCode = "missing_code"
	ErrInvalidRequest    ErrCode = "invalid_request"

	// validation detail errors
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
