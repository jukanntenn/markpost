package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"markpost/internal/infra"
	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestPostKey(t *testing.T) {
	t.Run("passes through with valid post key", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)

		created, _ := userRepo.Create(t.Context(), "test@example.com", "testuser", "password")

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.POST("/:post_key", PostKey(userRepo), func(c *gin.Context) {
			u, ok := ExtractUser(c)
			if !ok {
				c.String(http.StatusInternalServerError, "no user")
				return
			}
			c.JSON(http.StatusOK, gin.H{"user_id": u.ID})
		})

		req := httptest.NewRequest(http.MethodPost, "/"+created.PostKey, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["user_id"] != float64(created.ID) {
			t.Errorf("user_id = %v, want %d", resp["user_id"], created.ID)
		}
	})

	t.Run("rejects empty post key", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.POST("/", PostKey(userRepo), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("rejects invalid post key", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.POST("/:post_key", PostKey(userRepo), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodPost, "/invalid-key", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	// C3.1 漏洞补正（关键）：封禁用户即使持有有效 post_key 也拒绝发帖。
	t.Run("rejects post key of a disabled user (C3.1)", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		userRepo := infra.NewUserRepository(db, 16)

		created, _ := userRepo.Create(t.Context(), "banned@example.com", "banneduser", "password")
		_ = userRepo.SetActive(t.Context(), created.ID, false)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.POST("/:post_key", PostKey(userRepo), func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodPost, "/"+created.PostKey, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 for disabled user, got %d (body: %s)", w.Code, w.Body.String())
		}
		var resp struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != "user_disabled" {
			t.Errorf("expected code user_disabled, got %q", resp.Code)
		}
	})
}
