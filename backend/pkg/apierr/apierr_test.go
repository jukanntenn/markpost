package apierr

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"markpost/internal/service"
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
		{"invalid_credentials", service.NewServiceError(service.ErrInvalidCredentials, ""), http.StatusUnauthorized, "invalid_credentials"},
		{"invalid_password", service.NewServiceError(service.ErrInvalidPassword, ""), http.StatusBadRequest, "invalid_password"},
		{"not_found", service.NewServiceError(service.ErrNotFound, ""), http.StatusNotFound, "not_found"},
		{"unauthorized", service.NewServiceError(service.ErrUnauthorized, ""), http.StatusUnauthorized, "unauthorized"},
		{"failed_get_user", service.NewServiceError(service.ErrFailedGetUser, ""), http.StatusInternalServerError, "failed_get_user"},
		{"internal", service.NewServiceError(service.ErrInternal, ""), http.StatusInternalServerError, "internal"},
		{"validation", service.NewServiceError(service.ErrValidation, ""), http.StatusBadRequest, "validation"},
		{"invalid_request", service.NewServiceError(service.ErrInvalidRequest, ""), http.StatusBadRequest, "invalid_request"},
		{"missing_auth_header", service.NewServiceError(service.ErrMissingAuthorizationHeader, ""), http.StatusUnauthorized, "missing_authorization_header"},
		{"invalid_token", service.NewServiceError(service.ErrInvalidToken, ""), http.StatusUnauthorized, "invalid_token"},
		{"invalid_post_key", service.NewServiceError(service.ErrInvalidPostKey, ""), http.StatusForbidden, "invalid_post_key"},
		{"forbidden", service.NewServiceError(service.ErrForbidden, ""), http.StatusForbidden, "forbidden"},
		{"user_disabled", service.NewServiceError(service.ErrUserDisabled, ""), http.StatusForbidden, "user_disabled"},
		{"rate_limited", service.NewServiceError(service.ErrRateLimited, ""), http.StatusTooManyRequests, "rate_limited"},
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

func TestRespondError_UnknownCode(t *testing.T) {
	w := callRespondError(t, service.NewServiceError(service.ErrCode("nonexistent_code"), ""))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	resp := decodeErrorResponse(t, w.Body)
	if resp.Code != "nonexistent_code" {
		t.Errorf("code = %q, want %q", resp.Code, "nonexistent_code")
	}
	if resp.Message == "" {
		t.Error("message is empty")
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
	err := service.NewServiceErrorDetails(service.ErrValidation, "", []service.FieldDetail{
		{Code: service.ErrRequired, Description: "email"},
		{Code: service.ErrMinLength, Description: "password"},
	})

	w := callRespondError(t, err)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
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
	err := service.NewServiceError(service.ErrValidation, "")

	w := callRespondError(t, err)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	resp := decodeErrorResponse(t, w.Body)
	if len(resp.Errors) != 0 {
		t.Errorf("errors = %v, want empty", resp.Errors)
	}
}

func TestRespondError_ValidationUnknownFieldCode(t *testing.T) {
	unknownCode := service.ErrCode("nonexistent_field_code")
	err := service.NewServiceErrorDetails(service.ErrValidation, "", []service.FieldDetail{
		{Code: unknownCode, Description: "custom_field"},
	})

	w := callRespondError(t, err)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	resp := decodeErrorResponse(t, w.Body)
	if len(resp.Errors) != 1 {
		t.Fatalf("errors count = %d, want 1", len(resp.Errors))
	}

	if resp.Errors[0].Code != "nonexistent_field_code" {
		t.Errorf("errors[0].code = %q, want %q", resp.Errors[0].Code, "nonexistent_field_code")
	}
	if resp.Errors[0].Field != "custom_field" {
		t.Errorf("errors[0].field = %q, want %q", resp.Errors[0].Field, "custom_field")
	}
	if resp.Errors[0].Message == "" {
		t.Error("errors[0].message is empty")
	}
}
