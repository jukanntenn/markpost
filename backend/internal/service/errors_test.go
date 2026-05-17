package service

import (
	"errors"
	"testing"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		expected string
	}{
		{
			name:     "returns description when set",
			err:      Error{Description: "something"},
			expected: "something",
		},
		{
			name:     "returns wrapped error message when description is empty",
			err:      Error{Err: errors.New("wrapped")},
			expected: "wrapped",
		},
		{
			name:     "returns code string when both empty",
			err:      Error{Code: ErrValidation},
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
		e := Error{Err: inner}
		if got := e.Unwrap(); got == nil {
			t.Fatal("Unwrap() returned nil, want non-nil")
		} else if got.Error() != "inner" {
			t.Errorf("Unwrap().Error() = %q, want %q", got.Error(), "inner")
		}
	})

	t.Run("returns nil when no wrapped error", func(t *testing.T) {
		var e Error
		if got := e.Unwrap(); got != nil {
			t.Errorf("Unwrap() = %v, want nil", got)
		}
	})
}

func TestAsServiceError(t *testing.T) {
	t.Run("returns true for service error pointer", func(t *testing.T) {
		original := &Error{Code: ErrInternal}
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

func TestNewServiceErrorWithDetails(t *testing.T) {
	details := []Error{{Code: ErrRequired}}
	e := NewServiceErrorWithDetails(ErrValidation, "bad", details)
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
