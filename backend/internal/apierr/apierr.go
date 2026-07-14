// Package apierr is the single entry point for returning HTTP error responses
// from handlers and middleware. It translates a service.Error into the
// GitHub-style ErrorResponse, resolving the HTTP status and i18n message from
// the self-carrying service.ErrCode. See specs/backend/error-handling.md.
package apierr

import (
	"log/slog"

	"markpost/internal/service"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// FieldError represents a single validation error on a form field.
type FieldError struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents the JSON body returned when an API request fails.
type ErrorResponse struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
}

// RespondError writes a structured JSON error response to the gin context. It
// is the single entry point for handler/middleware error responses: a
// service.Error renders via its ErrCode's HTTP status + i18n message; any
// other error is logged (with trace context where available) and rendered as
// a 500 internal error. When the error is a validation error with field
// details, the per-field errors array is included.
func RespondError(c *gin.Context, err error) {
	se, ok := service.AsError(err)
	if !ok {
		slog.Error("unexpected error", "error", err,
			"method", c.Request.Method, "path", c.Request.URL.Path)
		writeError(c, service.ErrInternal, nil, nil)
		return
	}
	var fieldErrors []FieldError
	var data map[string]any
	if se.Code == service.ErrValidation {
		fieldErrors = buildFieldErrors(c, se.Details)
		if len(se.Details) > 0 {
			data = buildTemplateData(se)
		}
	}
	writeError(c, se.Code, data, fieldErrors)
}

// writeError renders an ErrorResponse at the code's HTTP status, with the
// message resolved via i18n and the optional field-level errors array.
func writeError(c *gin.Context, code *service.ErrCode, data map[string]any, fieldErrors []FieldError) {
	resp := ErrorResponse{
		Code:    code.Value,
		Message: renderMessage(c, code, data),
		Errors:  fieldErrors,
	}
	c.JSON(code.HTTP, resp)
}

// renderMessage resolves the i18n message for a code, falling back to the
// code's embedded English Other text when the locale lookup returns empty
// (the four-layer fallback in error-handling.md; never fails).
func renderMessage(c *gin.Context, code *service.ErrCode, data map[string]any) string {
	if code == nil || code.Message == nil {
		return ""
	}
	msg := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
		DefaultMessage: code.Message,
		TemplateData:   data,
	})
	if msg == "" {
		return code.Message.Other
	}
	return msg
}

// buildTemplateData constructs the i18n template data map from a service.Error,
// including the per-field Placeholder (Min/Max/...) and the universal Param.
func buildTemplateData(se *service.Error) map[string]any {
	if se == nil || len(se.Details) == 0 {
		return nil
	}
	// Field-validation templates use the first detail's Field + Param.
	d := se.Details[0]
	data := map[string]any{
		"Field": d.Field,
		"Param": d.Param,
	}
	if d.Code != nil && d.Code.Placeholder != "" {
		data[d.Code.Placeholder] = d.Param
	}
	return data
}

// buildFieldErrors renders the per-field errors array for an ErrValidation
// response, resolving each field's message via its own ErrCode i18n template.
func buildFieldErrors(c *gin.Context, details []service.FieldDetail) []FieldError {
	if len(details) == 0 {
		return nil
	}
	fieldErrors := make([]FieldError, 0, len(details))
	for _, d := range details {
		code := d.Code
		if code == nil {
			code = service.ErrFieldViolation
		}
		data := map[string]any{
			"Field": d.Field,
			"Param": d.Param,
		}
		if code.Placeholder != "" {
			data[code.Placeholder] = d.Param
		}
		fieldErrors = append(fieldErrors, FieldError{
			Field:   d.Field,
			Code:    code.Value,
			Message: renderMessage(c, code, data),
		})
	}
	return fieldErrors
}
