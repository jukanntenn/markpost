// Package service provides service-level error types and the shared error-code
// definitions consumed across the application. Per-domain error codes live in
// their own files (auth/errors.go, post/errors.go, ...); see error-handling.md.
package service

import (
	"encoding/json"
	"errors"
	"strconv"

	"markpost/internal/domain"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// ErrCode is a self-carrying error-code singleton: it bundles the
// machine-readable value, the HTTP status, the i18n message template (English
// fallback), and optional validator placeholder/param-provider. Instances are
// package-level vars passed by pointer; callers compare by .Value / .HTTP and
// never copy them.
type ErrCode struct {
	Value         string
	HTTP          int
	Message       *i18n.Message
	Placeholder   string
	ParamProvider func() string
}

// MarshalJSON renders an ErrCode as its string value, so ErrorResponse.Code is
// the machine-readable code regardless of how the ErrCode is embedded.
func (c *ErrCode) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte(`""`), nil
	}
	return jsonQuote(c.Value), nil
}

// FieldDetail describes a single field-level validation error. Field is the
// JSON/form field name; Code is the field error code (required/min/...); Param
// is the rule parameter (e.g. the min length) used by i18n template rendering.
type FieldDetail struct {
	Field string
	Code  *ErrCode
	Param string
}

// Error represents a service-level error. Code points at the authoritative
// ErrCode singleton (carrying HTTP/i18n mapping); Description is developer
// context (never sent to clients); Err is the wrapped underlying error; Details
// holds field-level validation errors for form binding.
type Error struct {
	Code        *ErrCode
	Description string
	Err         error
	Details     []FieldDetail
}

// Error returns the error message, preferring the description, then the wrapped
// error, then the code value.
func (e *Error) Error() string {
	if e.Description != "" {
		return e.Description
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Code != nil {
		return e.Code.Value
	}
	return ""
}

// Unwrap returns the underlying wrapped error.
func (e *Error) Unwrap() error {
	return e.Err
}

// AsError attempts to cast an error to *Error.
func AsError(err error) (*Error, bool) {
	var se *Error
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}

// New constructs an Error from a code and a developer description.
func New(code *ErrCode, description string) *Error {
	return &Error{Code: code, Description: description}
}

// Wrap constructs an Error that carries an underlying error.
func Wrap(code *ErrCode, description string, err error) *Error {
	return &Error{Code: code, Description: description, Err: err}
}

// WithDetails constructs an Error carrying field-level validation details.
func WithDetails(code *ErrCode, description string, details []FieldDetail) *Error {
	return &Error{Code: code, Description: description, Details: details}
}

// NewValidation is the convenience constructor for binding/validation errors:
// Code is ErrValidation and the details carry per-field causes.
func NewValidation(details []FieldDetail) *Error {
	return &Error{Code: ErrValidation, Description: "request validation failed", Details: details}
}

// WrapNotFoundOrInternal classifies an error as not-found (when it wraps
// domain.ErrNotFound) or internal otherwise.
func WrapNotFoundOrInternal(err error, notFoundMsg, internalMsg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrNotFound) {
		return Wrap(ErrNotFound, notFoundMsg, err)
	}
	return Wrap(ErrInternal, internalMsg, err)
}

// Shared error codes — common across all domains. Field-validation codes
// (ErrRequired, ErrMinLength, ...) return HTTP 422 per api-design.md §3.1
// (field validation failures are semantically "Unprocessable Entity").
var (
	ErrInternal = &ErrCode{
		Value:   "internal",
		HTTP:    500,
		Message: &i18n.Message{ID: "error.internal", Other: "Internal server error"},
	}
	ErrValidation = &ErrCode{
		Value:   "validation",
		HTTP:    422,
		Message: &i18n.Message{ID: "error.validation_failed", Other: "Request validation failed"},
	}
	ErrInvalidRequest = &ErrCode{
		Value:   "invalid_request",
		HTTP:    400,
		Message: &i18n.Message{ID: "error.invalid_request", Other: "Invalid request format"},
	}
	ErrNotFound = &ErrCode{
		Value:   "not_found",
		HTTP:    404,
		Message: &i18n.Message{ID: "error.not_found", Other: "Not Found"},
	}
	ErrUnauthorized = &ErrCode{
		Value:   "unauthorized",
		HTTP:    401,
		Message: &i18n.Message{ID: "error.unauthorized", Other: "Unauthorized"},
	}
	ErrForbidden = &ErrCode{
		Value:   "forbidden",
		HTTP:    403,
		Message: &i18n.Message{ID: "error.forbidden", Other: "Forbidden"},
	}
	ErrConflict = &ErrCode{
		Value:   "conflict",
		HTTP:    409,
		Message: &i18n.Message{ID: "error.conflict", Other: "Resource conflict"},
	}
	ErrRateLimited = &ErrCode{
		Value:   "rate_limited",
		HTTP:    429,
		Message: &i18n.Message{ID: "error.rate_limited", Other: "Too many requests"},
	}

	// Field-validation codes (all 422). Placeholder names the i18n template
	// field carrying the rule parameter (e.g. {{.Min}}); ParamProvider is used
	// by custom validators whose fe.Param() is empty.
	ErrRequired = &ErrCode{
		Value:   "required",
		HTTP:    422,
		Message: &i18n.Message{ID: "error.validation_required", Other: "{{.Field}} is required"},
	}
	ErrMinLength = &ErrCode{
		Value:       "min_length",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_min_length", Other: "{{.Field}} must be at least {{.Min}} characters"},
		Placeholder: "Min",
	}
	ErrMaxLength = &ErrCode{
		Value:       "max_length",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_max_length", Other: "{{.Field}} must be at most {{.Max}} characters"},
		Placeholder: "Max",
	}
	ErrLength = &ErrCode{
		Value:       "length",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_length", Other: "{{.Field}} must be {{.Len}} characters long"},
		Placeholder: "Len",
	}
	ErrEmail = &ErrCode{
		Value:   "invalid_email",
		HTTP:    422,
		Message: &i18n.Message{ID: "error.validation_email", Other: "{{.Field}} must be a valid email address"},
	}
	ErrOneOf = &ErrCode{
		Value:       "not_one_of",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_oneof", Other: "{{.Field}} must be one of: {{.OneOf}}"},
		Placeholder: "OneOf",
	}
	ErrFieldViolation = &ErrCode{
		Value:       "field_violation",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_field_violation", Other: "{{.Field}} is invalid"},
		Placeholder: "Param",
	}
)

// jsonQuote returns the JSON string encoding of s, used by ErrCode.MarshalJSON.
func jsonQuote(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(strconv.Quote(s))
	}
	return b
}
