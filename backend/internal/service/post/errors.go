package post

import "errors"

type ErrCode string

const (
	ErrNotFound ErrCode = "not_found"
	ErrInternal ErrCode = "internal"
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
