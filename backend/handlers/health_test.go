package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type healthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func TestHealth(t *testing.T) {
	t.Run("Success in English", testHealth_Success_EN)
	t.Run("Success in Chinese", testHealth_Success_ZH)
	t.Run("Success in default English", testHealth_Success_DefaultEN)
}

func testHealth_Success_EN(t *testing.T) {
	r := newTestI18nRouter(t)
	r.GET("/health", Health())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp.Status)
	}

	if resp.Message != "markpost is running" {
		t.Fatalf("expected message 'markpost is running', got %q", resp.Message)
	}
}

func testHealth_Success_ZH(t *testing.T) {
	r := newTestI18nRouter(t)
	r.GET("/health", Health())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp.Status)
	}

	if resp.Message != "markpost 正在运行" {
		t.Fatalf("expected message 'markpost 正在运行', got %q", resp.Message)
	}
}

func testHealth_Success_DefaultEN(t *testing.T) {
	r := newTestI18nRouter(t)
	r.GET("/health", Health())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp.Status)
	}

	if resp.Message != "markpost is running" {
		t.Fatalf("expected default message 'markpost is running', got %q", resp.Message)
	}
}
