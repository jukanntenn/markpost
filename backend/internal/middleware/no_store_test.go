package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNoStore(t *testing.T) {
	router := gin.New()
	router.Use(NoStore())
	router.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
	}
}
