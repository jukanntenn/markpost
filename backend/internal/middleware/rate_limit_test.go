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

func okHandler(c *gin.Context) { c.String(http.StatusOK, "ok") }

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

	t.Run("empty client IP yields 429", func(t *testing.T) {
		lmt := limiter.New(&limiter.ExpirableOptions{DefaultExpirationTTL: time.Minute})
		lmt.SetMax(10)
		lmt.SetBurst(10)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(RateLimitByIP(lmt))
		router.GET("/test", okHandler)

		// gin's ClientIP() returns "" when RemoteAddr is empty and no trusted
		// proxy header is present.
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ""
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("anonymous (empty IP) request: expected 429, got %d", w.Code)
		}
	})
}

func TestRateLimitByUserID(t *testing.T) {
	mkLimiter := func(max, burst float64) *limiter.Limiter {
		l := limiter.New(&limiter.ExpirableOptions{DefaultExpirationTTL: time.Minute})
		l.SetMax(max)
		l.SetBurst(int(burst))
		return l
	}

	withUser := func(id int) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("user_id", id)
			c.Next()
		}
	}

	t.Run("passes when under limit", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.GET("/w", withUser(5), RateLimitByUserID(mkLimiter(10, 10)), okHandler)

		req := httptest.NewRequest(http.MethodGet, "/w", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("rejects when over limit", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.GET("/w", withUser(5), RateLimitByUserID(mkLimiter(1, 1)), okHandler)

		req1 := httptest.NewRequest(http.MethodGet, "/w", nil)
		router.ServeHTTP(httptest.NewRecorder(), req1)

		req2 := httptest.NewRequest(http.MethodGet, "/w", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("over-limit request: expected 429, got %d", w2.Code)
		}
	})

	t.Run("isolates by user_id", func(t *testing.T) {
		// user 5 exhausts its bucket; user 6 must still pass — the limiters
		// are keyed on user_id, not shared globally.
		lmt := mkLimiter(1, 1)
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.GET("/w", func(c *gin.Context) {
			uid := 5
			if c.Query("u") == "6" {
				uid = 6
			}
			c.Set("user_id", uid)
			c.Next()
		}, RateLimitByUserID(lmt), okHandler)

		// Exhaust user 5's single token.
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/w?u=5", nil))
		wExhausted := httptest.NewRecorder()
		router.ServeHTTP(wExhausted, httptest.NewRequest(http.MethodGet, "/w?u=5", nil))
		if wExhausted.Code != http.StatusTooManyRequests {
			t.Fatalf("user 5 second request: expected 429, got %d", wExhausted.Code)
		}

		// user 6 still has its full budget.
		wOther := httptest.NewRecorder()
		router.ServeHTTP(wOther, httptest.NewRequest(http.MethodGet, "/w?u=6", nil))
		if wOther.Code != http.StatusOK {
			t.Errorf("user 6 first request: expected 200 (isolated), got %d", wOther.Code)
		}
	})

	t.Run("missing user_id yields 429", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		// No middleware sets user_id; the limiter must refuse rather than key
		// all anonymous requests together.
		router.GET("/w", RateLimitByUserID(mkLimiter(10, 10)), okHandler)

		req := httptest.NewRequest(http.MethodGet, "/w", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("missing user_id: expected 429, got %d", w.Code)
		}
	})

	t.Run("chains multiple limiters (L2 minute + daily)", func(t *testing.T) {
		minute := mkLimiter(1, 1) // 1 req allowed, then blocked
		daily := mkLimiter(10, 10)
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.GET("/w", withUser(5), RateLimitByUserID(minute, daily), okHandler)

		// First passes both.
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/w", nil))
		if w1.Code != http.StatusOK {
			t.Fatalf("first chained request: expected 200, got %d", w1.Code)
		}
		// Second is blocked by the minute limiter even though the daily one
		// still has budget.
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/w", nil))
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("second chained request: expected 429 from minute limiter, got %d", w2.Code)
		}
	})
}

func TestRateLimiters_L1AndL2AreIsolated(t *testing.T) {
	// L1 (IP) and L2 (user_id) are independent limiters; exhausting one must
	// not affect the other because they key on different dimensions.
	l1 := NewLimiter(1, 1)
	l2 := NewLimiter(1, 1)

	router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
	router.GET("/read", RateLimitByIP(l1), okHandler)
	router.GET("/write", func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Next()
	}, RateLimitByUserID(l2), okHandler)

	// Exhaust L1 (IP) for the read path.
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/read", nil))
	wReadBlocked := httptest.NewRecorder()
	router.ServeHTTP(wReadBlocked, httptest.NewRequest(http.MethodGet, "/read", nil))
	if wReadBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("read path should be blocked after exhausting L1, got %d", wReadBlocked.Code)
	}

	// The write path (L2, different dimension) is unaffected.
	wWrite := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/write", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(wWrite, req)
	if wWrite.Code != http.StatusOK {
		t.Errorf("write path should be unaffected by L1 exhaustion, got %d", wWrite.Code)
	}
}
