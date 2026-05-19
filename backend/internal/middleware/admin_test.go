package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"markpost/internal/domain/user"
	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestExtractUser(t *testing.T) {
	t.Run("returns user when present", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expected := &user.User{ID: 1, Role: user.RoleAdmin}
		c.Set("user", expected)

		u, ok := ExtractUser(c)
		if !ok {
			t.Fatal("expected ok to be true")
		}
		if u.ID != 1 {
			t.Errorf("expected ID 1, got %d", u.ID)
		}
		if u.Role != user.RoleAdmin {
			t.Errorf("expected role admin, got %s", u.Role)
		}
	})

	t.Run("returns false when missing", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		u, ok := ExtractUser(c)
		if ok {
			t.Fatal("expected ok to be false")
		}
		if u != nil {
			t.Error("expected nil user")
		}
	})

	t.Run("returns false when wrong type", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Set("user", "not a user struct")

		u, ok := ExtractUser(c)
		if ok {
			t.Fatal("expected ok to be false")
		}
		if u != nil {
			t.Error("expected nil user")
		}
	})
}

func TestRequireAdmin(t *testing.T) {
	t.Run("aborts with 401 when no user", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{
			LocalesPath: "../../locales",
		})
		router.Use(RequireAdmin())
		router.GET("/admin", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["code"] != "unauthorized" {
			t.Errorf("expected code 'unauthorized', got %v", resp["code"])
		}
	})

	t.Run("aborts with 403 when non-admin", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{
			LocalesPath: "../../locales",
		})
		router.Use(func(c *gin.Context) {
			c.Set("user", &user.User{ID: 1, Role: user.RoleUser})
		})
		router.Use(RequireAdmin())
		router.GET("/admin", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["code"] != "forbidden" {
			t.Errorf("expected code 'forbidden', got %v", resp["code"])
		}
	})

	t.Run("passes through for admin user", func(t *testing.T) {
		router := testutil.NewTestEngine(testutil.TestEngineConfig{
			LocalesPath: "../../locales",
		})
		router.Use(func(c *gin.Context) {
			c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		})
		router.Use(RequireAdmin())
		router.GET("/admin", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}
