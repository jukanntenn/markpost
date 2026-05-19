package service

import (
	"errors"
	"testing"
)

func TestErrCode_String(t *testing.T) {
	t.Run("returns underlying string", func(t *testing.T) {
		if got := ErrValidation.String(); got != "validation" {
			t.Errorf("ErrValidation.String() = %q, want %q", got, "validation")
		}
	})
}

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ServiceError
		expected string
	}{
		{
			name:     "returns description when set",
			err:      ServiceError{Description: "something"},
			expected: "something",
		},
		{
			name:     "returns wrapped error message when description is empty",
			err:      ServiceError{Err: errors.New("wrapped")},
			expected: "wrapped",
		},
		{
			name:     "returns description when both description and err are set",
			err:      ServiceError{Description: "desc", Err: errors.New("wrapped")},
			expected: "desc",
		},
		{
			name:     "returns code string when both empty",
			err:      ServiceError{Code: ErrValidation},
			expected: "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	t.Run("returns wrapped error", func(t *testing.T) {
		inner := errors.New("inner")
		e := ServiceError{Err: inner}
		if got := e.Unwrap(); got == nil {
			t.Fatal("Unwrap() returned nil, want non-nil")
		} else if got.Error() != "inner" {
			t.Errorf("Unwrap().Error() = %q, want %q", got.Error(), "inner")
		}
	})

	t.Run("returns nil when no wrapped error", func(t *testing.T) {
		var e ServiceError
		if got := e.Unwrap(); got != nil {
			t.Errorf("Unwrap() = %v, want nil", got)
		}
	})
}

func TestAsServiceError(t *testing.T) {
	t.Run("returns true for service error pointer", func(t *testing.T) {
		original := &ServiceError{Code: ErrInternal}
		se, ok := AsServiceError(original)
		if !ok {
			t.Fatal("AsServiceError() returned false, want true")
		}
		if se != original {
			t.Error("returned pointer does not point to the same Error value")
		}
	})

	t.Run("returns false for non-service error", func(t *testing.T) {
		se, ok := AsServiceError(errors.New("plain"))
		if ok {
			t.Fatal("AsServiceError() returned true for plain error, want false")
		}
		if se != nil {
			t.Errorf("AsServiceError() = %v, want nil", se)
		}
	})
}

func TestNewServiceError(t *testing.T) {
	e := NewServiceError(ErrValidation, "bad")
	if e.Code != ErrValidation {
		t.Errorf("Code = %q, want %q", e.Code, ErrValidation)
	}
	if e.Description != "bad" {
		t.Errorf("Description = %q, want %q", e.Description, "bad")
	}
	if e.Err != nil {
		t.Errorf("Err = %v, want nil", e.Err)
	}
	if e.Details != nil {
		t.Errorf("Details = %v, want nil", e.Details)
	}
}

func TestNewServiceErrorWrap(t *testing.T) {
	cause := errors.New("cause")
	e := NewServiceErrorWrap(ErrInternal, "msg", cause)
	if e.Code != ErrInternal {
		t.Errorf("Code = %q, want %q", e.Code, ErrInternal)
	}
	if e.Description != "msg" {
		t.Errorf("Description = %q, want %q", e.Description, "msg")
	}
	if e.Err != cause {
		t.Errorf("Err = %v, want %v", e.Err, cause)
	}
	if e.Details != nil {
		t.Errorf("Details = %v, want nil", e.Details)
	}
}

func TestError_HTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrCode
		expected int
	}{
		{"invalid_credentials returns 401", ErrInvalidCredentials, 401},
		{"invalid_password returns 400", ErrInvalidPassword, 400},
		{"not_found returns 404", ErrNotFound, 404},
		{"unauthorized returns 401", ErrUnauthorized, 401},
		{"internal returns 500", ErrInternal, 500},
		{"validation returns 400", ErrValidation, 400},
		{"forbidden returns 403", ErrForbidden, 403},
		{"rate_limited returns 429", ErrRateLimited, 429},
		{"required returns 400", ErrRequired, 400},
		{"min_length returns 400", ErrMinLength, 400},
		{"field_violation returns 400", ErrFieldViolation, 400},
		{"unknown code defaults to 500", ErrCode("nonexistent"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ServiceError{Code: tt.code}
			if got := e.HTTPStatus(); got != tt.expected {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

var allErrCodes = []ErrCode{
	ErrInvalidCredentials,
	ErrInvalidPassword,
	ErrUnauthorized,
	ErrInternal,
	ErrValidation,
	ErrNotFound,
	ErrFailedGetUser,
	ErrMissingAuthorizationHeader,
	ErrInvalidToken,
	ErrInvalidPostKey,
	ErrUserDisabled,
	ErrForbidden,
	ErrRateLimited,
	ErrInvalidRequest,
	ErrRequired,
	ErrMinLength,
	ErrFieldViolation,
}

func TestHTTPStatusesCompleteness(t *testing.T) {
	for _, code := range allErrCodes {
		_, ok := httpStatuses[code]
		if !ok {
			t.Errorf("ErrCode %q missing from httpStatuses", code)
		}
	}
	if len(allErrCodes) != len(httpStatuses) {
		t.Errorf("httpStatuses has %d entries, expected %d (allErrCodes)", len(httpStatuses), len(allErrCodes))
	}
}

func TestNewServiceErrorDetails(t *testing.T) {
	details := []FieldDetail{{Code: ErrRequired}}
	e := NewServiceErrorDetails(ErrValidation, "bad", details)
	if e.Code != ErrValidation {
		t.Errorf("Code = %q, want %q", e.Code, ErrValidation)
	}
	if e.Description != "bad" {
		t.Errorf("Description = %q, want %q", e.Description, "bad")
	}
	if len(e.Details) != 1 || e.Details[0].Code != ErrRequired {
		t.Errorf("Details = %v, want [{Code: required}]", e.Details)
	}
	if e.Err != nil {
		t.Errorf("Err = %v, want nil", e.Err)
	}
}
