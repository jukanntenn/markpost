package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestFallback(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Fallback())
		router.GET("/panic", func(_ *gin.Context) {
			panic("something went wrong")
		})

		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("passes through without panic", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{})
		router.Use(Fallback())
		router.GET("/ok", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}
