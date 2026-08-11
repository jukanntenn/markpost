package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service/auth"
	"markpost/internal/testutil"
	"markpost/pkg/utils"

	"github.com/gin-gonic/gin"
)

func setupAuthMiddleware(t *testing.T) (*auth.JWTService, user.Repository, user.TokenRepository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	jwtSvc := auth.NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
	return jwtSvc, userRepo, tokenRepo
}

func TestAuth(t *testing.T) {
	t.Run("passes through with valid token", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		created, _ := userRepo.Create(t.Context(), "test@example.com", "testuser", "password")
		_ = userRepo.SetRole(t.Context(), created.ID, user.RoleUser)

		token, _ := jwtSvc.GenerateAccessToken(time.Now(), created.ID, "test@example.com", "testuser", "user", 0)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Auth(jwtSvc, userRepo))
		router.GET("/protected", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("rejects missing Authorization header", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Auth(jwtSvc, userRepo))
		router.GET("/protected", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Auth(jwtSvc, userRepo))
		router.GET("/protected", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("rejects token for deleted user", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		created, _ := userRepo.Create(t.Context(), "test@example.com", "testuser", "password")
		token, _ := jwtSvc.GenerateAccessToken(time.Now(), created.ID, "test@example.com", "testuser", "user", 0)

		_, _ = userRepo.DeleteByID(t.Context(), created.ID)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Auth(jwtSvc, userRepo))
		router.GET("/protected", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// K.4 fail-close：user 查询失败（含用户已删除）一律 401，宁误杀不放行。
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestAuthWithBlacklist(t *testing.T) {
	t.Run("rejects blacklisted token", func(t *testing.T) {
		jwtSvc, userRepo, tokenRepo := setupAuthMiddleware(t)

		created, _ := userRepo.Create(t.Context(), "test@example.com", "testuser", "password")
		token, _ := jwtSvc.GenerateAccessToken(time.Now(), created.ID, "test@example.com", "testuser", "user", 0)

		tokenHash := utils.HashToken(token)
		_ = tokenRepo.StoreBlacklistedToken(t.Context(), tokenHash, time.Now().Add(time.Hour))

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(AuthWithBlacklist(jwtSvc, userRepo, tokenRepo))
		router.GET("/protected", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["code"] != "invalid_token" {
			t.Errorf("expected code 'invalid_token', got %v", resp["code"])
		}
	})
}

func TestExtractAccessToken(t *testing.T) {
	t.Run("returns token when set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("access_token", "my-token")

		token, ok := ExtractAccessToken(c)
		if !ok {
			t.Fatal("expected ok to be true")
		}
		if token != "my-token" {
			t.Errorf("expected 'my-token', got %q", token)
		}
	})

	t.Run("returns false when not set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		_, ok := ExtractAccessToken(c)
		if ok {
			t.Error("expected ok to be false")
		}
	})
}

func TestOptionalAuth(t *testing.T) {
	t.Run("sets user when valid token provided", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		created, _ := userRepo.Create(t.Context(), "test@example.com", "testuser", "password")
		token, _ := jwtSvc.GenerateAccessToken(time.Now(), created.ID, "test@example.com", "testuser", "user", 0)

		var gotUser *user.User
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(OptionalAuth(jwtSvc, userRepo))
		router.GET("/optional", func(c *gin.Context) {
			gotUser, _ = ExtractUser(c)
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		if gotUser == nil {
			t.Error("expected user to be set")
		}
	})

	t.Run("continues without user when no token", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(OptionalAuth(jwtSvc, userRepo))
		router.GET("/optional", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// C2.6：token version 即时失效 —— 改密/强制下线/封禁后旧 access token 被拒。
func TestAuth_TokenVersion(t *testing.T) {
	t.Run("rejects token with stale token_version (C2.6)", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)

		created, _ := userRepo.Create(t.Context(), "tv@example.com", "tvuser", "password")

		// 用户当前 token_version=0，签发 tv=0 的 token → 通过。
		token, _ := jwtSvc.GenerateAccessToken(time.Now(), created.ID, "tv@example.com", "tvuser", "user", 0)
		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Auth(jwtSvc, userRepo))
		router.GET("/protected", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 with matching tv, got %d", w.Code)
		}

		// 模拟强制下线：token_version++ → 旧 token 立即失效。
		if err := userRepo.BumpTokenVersion(t.Context(), created.ID); err != nil {
			t.Fatalf("bump token version: %v", err)
		}

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 with stale tv, got %d", w.Code)
		}
	})

	t.Run("rejects disabled user regardless of tv (user_disabled)", func(t *testing.T) {
		jwtSvc, userRepo, _ := setupAuthMiddleware(t)
		created, _ := userRepo.Create(t.Context(), "d@example.com", "duser", "password")
		token, _ := jwtSvc.GenerateAccessToken(time.Now(), created.ID, "d@example.com", "duser", "user", 0)

		router := testutil.NewTestEngine(testutil.TestEngineConfig{LocalesPath: "../../locales"})
		router.Use(Auth(jwtSvc, userRepo))
		router.GET("/protected", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 before ban, got %d", w.Code)
		}

		_ = userRepo.SetActive(t.Context(), created.ID, false)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 for disabled user, got %d", w.Code)
		}
	})
}
