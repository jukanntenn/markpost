package apierr

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"markpost/internal/service"
	"markpost/internal/service/auth"
	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
)

func callRespondError(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()
	router := testutil.NewTestEngine(testutil.TestEngineConfig{
		LocalesPath: "../../locales",
	})
	router.GET("/test", func(c *gin.Context) {
		RespondError(c, err)
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeErrorResponse(t *testing.T, body io.Reader) ErrorResponse {
	t.Helper()
	var resp ErrorResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return resp
}

func TestRespondError_MappedCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid_credentials", service.New(auth.ErrInvalidCredentials, ""), http.StatusUnauthorized, "invalid_credentials"},
		{"invalid_password", service.New(auth.ErrInvalidPassword, ""), http.StatusUnauthorized, "invalid_password"},
		{"not_found", service.New(service.ErrNotFound, ""), http.StatusNotFound, "not_found"},
		{"unauthorized", service.New(service.ErrUnauthorized, ""), http.StatusUnauthorized, "unauthorized"},
		{"internal", service.New(service.ErrInternal, ""), http.StatusInternalServerError, "internal"},
		{"validation", service.New(service.ErrValidation, ""), http.StatusUnprocessableEntity, "validation"},
		{"invalid_request", service.New(service.ErrInvalidRequest, ""), http.StatusBadRequest, "invalid_request"},
		{"invalid_token", service.New(auth.ErrInvalidToken, ""), http.StatusUnauthorized, "invalid_token"},
		{"invalid_post_key", service.New(auth.ErrInvalidPostKey, ""), http.StatusForbidden, "invalid_post_key"},
		{"forbidden", service.New(service.ErrForbidden, ""), http.StatusForbidden, "forbidden"},
		{"user_disabled", service.New(auth.ErrUserDisabled, ""), http.StatusForbidden, "user_disabled"},
		{"rate_limited", service.New(service.ErrRateLimited, ""), http.StatusTooManyRequests, "rate_limited"},
		{"conflict", service.New(service.ErrConflict, ""), http.StatusConflict, "conflict"},
		{"github_user_fetch", service.New(auth.ErrGitHubUserFetch, ""), http.StatusBadGateway, "github_user_fetch_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := callRespondError(t, tt.err)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			resp := decodeErrorResponse(t, w.Body)
			if resp.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", resp.Code, tt.wantCode)
			}
			if resp.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

func TestRespondError_NonServiceError(t *testing.T) {
	w := callRespondError(t, errors.New("something"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	resp := decodeErrorResponse(t, w.Body)
	if resp.Code != "internal" {
		t.Errorf("code = %q, want %q", resp.Code, "internal")
	}
}

func TestRespondError_ValidationWithDetails(t *testing.T) {
	err := service.WithDetails(service.ErrValidation, "", []service.FieldDetail{
		{Field: "email", Code: service.ErrRequired},
		{Field: "password", Code: service.ErrMinLength, Param: "8"},
	})

	w := callRespondError(t, err)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}

	resp := decodeErrorResponse(t, w.Body)
	if len(resp.Errors) != 2 {
		t.Fatalf("errors count = %d, want 2", len(resp.Errors))
	}

	if resp.Errors[0].Code != "required" {
		t.Errorf("errors[0].code = %q, want %q", resp.Errors[0].Code, "required")
	}
	if resp.Errors[0].Field != "email" {
		t.Errorf("errors[0].field = %q, want %q", resp.Errors[0].Field, "email")
	}
	if resp.Errors[0].Message == "" {
		t.Error("errors[0].message is empty")
	}

	if resp.Errors[1].Code != "min_length" {
		t.Errorf("errors[1].code = %q, want %q", resp.Errors[1].Code, "min_length")
	}
	if resp.Errors[1].Field != "password" {
		t.Errorf("errors[1].field = %q, want %q", resp.Errors[1].Field, "password")
	}
	if resp.Errors[1].Message == "" {
		t.Error("errors[1].message is empty")
	}
}

func TestRespondError_ValidationEmptyDetails(t *testing.T) {
	err := service.New(service.ErrValidation, "")

	w := callRespondError(t, err)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}

	resp := decodeErrorResponse(t, w.Body)
	if len(resp.Errors) != 0 {
		t.Errorf("errors = %v, want empty", resp.Errors)
	}
}
