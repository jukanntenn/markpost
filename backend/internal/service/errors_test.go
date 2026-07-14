package service

import (
	"errors"
	"testing"

	"markpost/internal/domain"
)

func TestErrCode_Value(t *testing.T) {
	if got := ErrValidation.Value; got != "validation" {
		t.Errorf("ErrValidation.Value = %q, want %q", got, "validation")
	}
}

func TestError_Error(t *testing.T) {
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
			name:     "returns description when both description and err are set",
			err:      Error{Description: "desc", Err: errors.New("wrapped")},
			expected: "desc",
		},
		{
			name:     "returns code value when both empty",
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

func TestError_Unwrap(t *testing.T) {
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

func TestAsError(t *testing.T) {
	t.Run("returns true for service error pointer", func(t *testing.T) {
		original := &Error{Code: ErrInternal}
		se, ok := AsError(original)
		if !ok {
			t.Fatal("AsError() returned false, want true")
		}
		if se != original {
			t.Error("returned pointer does not point to the same Error value")
		}
	})

	t.Run("returns false for non-service error", func(t *testing.T) {
		se, ok := AsError(errors.New("plain"))
		if ok {
			t.Fatal("AsError() returned true for plain error, want false")
		}
		if se != nil {
			t.Errorf("AsError() = %v, want nil", se)
		}
	})
}

func TestNew(t *testing.T) {
	e := New(ErrValidation, "bad")
	if e.Code != ErrValidation {
		t.Errorf("Code = %p, want %p", e.Code, ErrValidation)
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

func TestWrap(t *testing.T) {
	cause := errors.New("cause")
	e := Wrap(ErrInternal, "msg", cause)
	if e.Code != ErrInternal {
		t.Errorf("Code mismatch")
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

func TestErrCode_HTTP(t *testing.T) {
	tests := []struct {
		name     string
		code     *ErrCode
		expected int
	}{
		{"not_found returns 404", ErrNotFound, 404},
		{"unauthorized returns 401", ErrUnauthorized, 401},
		{"internal returns 500", ErrInternal, 500},
		{"validation returns 422", ErrValidation, 422},
		{"forbidden returns 403", ErrForbidden, 403},
		{"rate_limited returns 429", ErrRateLimited, 429},
		{"invalid_request returns 400", ErrInvalidRequest, 400},
		{"conflict returns 409", ErrConflict, 409},
		{"required returns 422", ErrRequired, 422},
		{"min_length returns 422", ErrMinLength, 422},
		{"max_length returns 422", ErrMaxLength, 422},
		{"field_violation returns 422", ErrFieldViolation, 422},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.HTTP; got != tt.expected {
				t.Errorf("HTTP = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestWithDetails(t *testing.T) {
	details := []FieldDetail{{Code: ErrRequired}}
	e := WithDetails(ErrValidation, "bad", details)
	if e.Code != ErrValidation {
		t.Errorf("Code mismatch")
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

func TestWrapNotFoundOrInternal(t *testing.T) {
	t.Run("returns nil for nil error", func(t *testing.T) {
		err := WrapNotFoundOrInternal(nil, "not found", "internal")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("wraps ErrNotFound", func(t *testing.T) {
		err := WrapNotFoundOrInternal(domain.ErrNotFound, "not found msg", "internal msg")
		se, ok := AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrNotFound {
			t.Errorf("code mismatch")
		}
		if se.Description != "not found msg" {
			t.Errorf("description = %q, want %q", se.Description, "not found msg")
		}
	})

	t.Run("wraps other errors as internal", func(t *testing.T) {
		err := WrapNotFoundOrInternal(errors.New("db error"), "not found msg", "internal msg")
		se, ok := AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != ErrInternal {
			t.Errorf("code mismatch")
		}
		if se.Description != "internal msg" {
			t.Errorf("description = %q, want %q", se.Description, "internal msg")
		}
	})
}

func TestNewValidation(t *testing.T) {
	details := []FieldDetail{{Field: "title", Code: ErrRequired}}
	e := NewValidation(details)
	if e.Code != ErrValidation {
		t.Errorf("Code mismatch")
	}
	if e.Description != "request validation failed" {
		t.Errorf("Description = %q, want %q", e.Description, "request validation failed")
	}
	if len(e.Details) != 1 {
		t.Errorf("expected 1 detail, got %d", len(e.Details))
	}
}

func TestErrCode_MarshalJSON(t *testing.T) {
	t.Run("renders value as JSON string", func(t *testing.T) {
		b, err := ErrValidation.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if got := string(b); got != `"validation"` {
			t.Errorf("MarshalJSON = %s, want %q", got, `"validation"`)
		}
	})
	t.Run("nil renders empty string", func(t *testing.T) {
		var c *ErrCode
		b, err := c.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if got := string(b); got != `""` {
			t.Errorf("MarshalJSON = %s, want %q", got, `""`)
		}
	})
}
