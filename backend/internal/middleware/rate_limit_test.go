package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/testutil"

	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
)

func TestRateLimitByIP(t *testing.T) {
	t.Run("passes through when under limit", func(t *testing.T) {
		lmt := limiter.New(&limiter.ExpirableOptions{
			DefaultExpirationTTL: time.Minute,
		})
		lmt.SetMax(10)
		lmt.SetBurst(10)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(RateLimitByIP(lmt))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("rejects when over limit", func(t *testing.T) {
		lmt := limiter.New(&limiter.ExpirableOptions{
			DefaultExpirationTTL: time.Minute,
		})
		lmt.SetMax(1)
		lmt.SetBurst(1)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(RateLimitByIP(lmt))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// First request should pass
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.RemoteAddr = "192.168.1.1:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		if w1.Code != http.StatusOK {
			t.Errorf("first request: expected status %d, got %d", http.StatusOK, w1.Code)
		}

		// Second request should be rate limited
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.RemoteAddr = "192.168.1.1:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("second request: expected status %d, got %d", http.StatusTooManyRequests, w2.Code)
		}
	})
}
