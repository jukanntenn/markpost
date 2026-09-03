package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// FieldError is one entry of the server's field-error list (binding and
// validation failures carry per-field codes).
type FieldError struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPError is a decoded API error envelope: {code, message, errors?}. The
// main package prints its message; commands may branch on StatusCode.
type HTTPError struct {
	StatusCode  int
	Code        string
	Message     string
	FieldErrors []FieldError
}

func (e *HTTPError) Error() string {
	msg := fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	if e.Code != "" && e.Code != e.Message {
		msg = fmt.Sprintf("HTTP %d: %s: %s", e.StatusCode, e.Code, e.Message)
	}
	if len(e.FieldErrors) > 0 {
		parts := make([]string, 0, len(e.FieldErrors))
		for _, fe := range e.FieldErrors {
			if fe.Field != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", fe.Field, fe.Code))
			} else {
				parts = append(parts, fe.Code)
			}
		}
		msg += " (" + strings.Join(parts, ", ") + ")"
	}
	return msg
}

func newHTTPErrorStatus(status int, body []byte) *HTTPError {
	e := &HTTPError{StatusCode: status}
	if len(body) > 0 {
		var env struct {
			Code    string       `json:"code"`
			Message string       `json:"message"`
			Errors  []FieldError `json:"errors"`
		}
		if json.Unmarshal(body, &env) == nil {
			e.Code = env.Code
			e.Message = env.Message
			e.FieldErrors = env.Errors
		}
	}
	if e.Message == "" {
		e.Message = http.StatusText(status)
	}
	return e
}

// AuthError marks a session-level failure the user fixes by logging in again:
// no stored credentials, or a 401 that token refresh could not cure. It maps
// to exit code 4 (gh's exitAuth), distinct from ordinary API errors.
type AuthError struct {
	Message string
	Cause   error
}

func (e *AuthError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *AuthError) Unwrap() error { return e.Cause }

// AsHTTPError extracts an HTTPError if err's chain carries one.
func AsHTTPError(err error) (*HTTPError, bool) {
	var httpErr *HTTPError
	return httpErr, errors.As(err, &httpErr)
}

// IsAuthError reports whether err's chain carries an AuthError.
func IsAuthError(err error) bool {
	var authErr *AuthError
	return errors.As(err, &authErr)
}
