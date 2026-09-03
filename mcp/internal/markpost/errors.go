package markpost

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// FieldError mirrors one entry of the backend's 422 validation "errors" array.
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
	Param string `json:"param,omitempty"`
}

// errorBody mirrors the backend's GitHub-style error envelope
// (backend/internal/apierr): {"error": {"code", "message", "errors"?}}.
type errorBody struct {
	Error struct {
		Code    string       `json:"code"`
		Message string       `json:"message"`
		Errors  []FieldError `json:"errors"`
	} `json:"error"`
}

// APIError is a non-2xx markpost response. Callers surface Code and Message
// to the agent verbatim — they carry the backend's semantics.
type APIError struct {
	StatusCode  int
	Code        string
	Message     string
	FieldErrors []FieldError
}

func (e *APIError) Error() string {
	if len(e.FieldErrors) > 0 {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.FieldErrors)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func apiError(status int, body []byte) error {
	var eb errorBody
	if err := json.Unmarshal(body, &eb); err == nil && eb.Error.Code != "" {
		return &APIError{
			StatusCode:  status,
			Code:        eb.Error.Code,
			Message:     eb.Error.Message,
			FieldErrors: eb.Error.Errors,
		}
	}
	return &APIError{
		StatusCode: status,
		Code:       fmt.Sprintf("http_%d", status),
		Message:    fmt.Sprintf("unexpected non-JSON error response (HTTP %d): %.200s", status, body),
	}
}

// NotFound reports whether err is a 404 from the markpost API.
func NotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
