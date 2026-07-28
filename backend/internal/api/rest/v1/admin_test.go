package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"markpost/internal/domain/audit"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service/admin"

	"github.com/gin-gonic/gin"
)

func setupAdminHandlerDeps(t *testing.T) (*admin.Service, user.Repository, post.Repository, delivery.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	postRepo := infra.NewPostRepository(db)
	channelRepo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)
	sessionLister := &mockSessionLister{}
	auditRecorder := &mockAuditRecorder{}

	svc := admin.NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
		sessionLister,
		auditRecorder,
	)
	return svc, userRepo, postRepo, channelRepo
}

type postListerAdapter struct {
	repo post.Repository
}

func (a *postListerAdapter) GetAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error) {
	items, err := a.repo.ListAll(ctx, search, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	count, err := a.repo.CountAll(ctx, search)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

type channelListerAdapter struct {
	repo delivery.Repository
}

func (a *channelListerAdapter) ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error) {
	items, err := a.repo.ListAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	count, err := a.repo.CountAll(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

type mockAuditRecorder struct {
	logs []audit.Log
}

func (m *mockAuditRecorder) Record(ctx context.Context, e audit.Entry) error {
	m.logs = append(m.logs, audit.Log{
		ActorID:    e.ActorID,
		Action:     e.Action,
		TargetType: e.TargetType,
		TargetID:   e.TargetID,
	})
	return nil
}

func (m *mockAuditRecorder) List(ctx context.Context, offset, limit int) ([]audit.Log, int64, error) {
	total := int64(len(m.logs))
	if offset >= len(m.logs) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(m.logs) {
		end = len(m.logs)
	}
	return m.logs[offset:end], total, nil
}

type mockSessionLister struct{}

func (m *mockSessionLister) ListByUserID(ctx context.Context, userID int) ([]user.RefreshToken, error) {
	return nil, nil
}

func (m *mockSessionLister) RevokeAllByUserID(ctx context.Context, userID int) error {
	return nil
}

func TestAdminListUsers_Success(t *testing.T) {
	svc, userRepo, _, _ := setupAdminHandlerDeps(t)
	ctx := t.Context()
	_, _ = userRepo.Create(ctx, "a@example.com", "alice", "pass")
	_, _ = userRepo.Create(ctx, "b@example.com", "bob", "pass")

	router := newTestEngine()
	router.GET("/admin/users", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListUsers(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	users, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("expected users array in response")
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestAdminListUsers_WithPagination(t *testing.T) {
	svc, userRepo, _, _ := setupAdminHandlerDeps(t)
	ctx := t.Context()
	_, _ = userRepo.Create(ctx, "a@example.com", "alice", "pass")
	_, _ = userRepo.Create(ctx, "b@example.com", "bob", "pass")
	_, _ = userRepo.Create(ctx, "c@example.com", "charlie", "pass")

	router := newTestEngine()
	router.GET("/admin/users", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListUsers(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/users?page=1&limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	total, ok := resp["total"].(float64)
	if !ok {
		t.Fatal("expected total in response")
	}
	if int(total) != 3 {
		t.Errorf("expected total 3, got %v", total)
	}
}

func TestAdminListPosts_Success(t *testing.T) {
	svc, userRepo, postRepo, _ := setupAdminHandlerDeps(t)
	ctx := t.Context()
	u1, _ := userRepo.Create(ctx, "user1@example.com", "user1", "password")
	u2, _ := userRepo.Create(ctx, "user2@example.com", "user2", "password")
	_, _ = postRepo.Create(ctx, "Post 1", "Body 1", u1.ID)
	_, _ = postRepo.Create(ctx, "Post 2", "Body 2", u2.ID)

	router := newTestEngine()
	router.GET("/admin/posts", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListPosts(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/posts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	posts, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("expected posts array in response")
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}
}

func TestAdminListPosts_WithSearch(t *testing.T) {
	svc, userRepo, postRepo, _ := setupAdminHandlerDeps(t)
	ctx := t.Context()
	u1, _ := userRepo.Create(ctx, "user1@example.com", "user1", "password")
	u2, _ := userRepo.Create(ctx, "user2@example.com", "user2", "password")
	_, _ = postRepo.Create(ctx, "Alert Post", "Body", u1.ID)
	_, _ = postRepo.Create(ctx, "Normal Post", "Body", u2.ID)

	router := newTestEngine()
	router.GET("/admin/posts", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListPosts(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/posts?search=Alert", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	posts, _ := resp["items"].([]interface{})
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
}

func TestAdminListChannels_Success(t *testing.T) {
	svc, userRepo, _, channelRepo := setupAdminHandlerDeps(t)
	ctx := t.Context()
	u1, _ := userRepo.Create(ctx, "user1@example.com", "user1", "password")
	u2, _ := userRepo.Create(ctx, "user2@example.com", "user2", "password")
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: u1.ID, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: u2.ID, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://b.com", "card_link_url": ""}})

	router := newTestEngine()
	router.GET("/admin/channels", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListChannels(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/channels", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	channels, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("expected channels array in response")
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}

func setupAdminHandlerWithMutators(t *testing.T) (*admin.Service, user.Repository, delivery.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	postRepo := infra.NewPostRepository(db)
	channelRepo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)
	sessionLister := &mockSessionLister{}
	auditRecorder := &mockAuditRecorder{}

	svc := admin.NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
		sessionLister,
		auditRecorder,
	)
	svc.SetUserMutator(userRepo.(*infra.UserRepository))
	svc.SetChannelMutator(channelRepo)
	return svc, userRepo, channelRepo
}

func TestAdminCreateUser_Success(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.POST("/admin/users", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminCreateUser(svc))

	body := `{"email":"new@example.com","username":"newuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp AdminUserItem
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Email != "new@example.com" {
		t.Errorf("email = %q, want %q", resp.Email, "new@example.com")
	}
}

func TestAdminCreateUser_InvalidBody(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.POST("/admin/users", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminCreateUser(svc))

	t.Run("missing email", func(t *testing.T) {
		body := `{"username":"user","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
			t.Errorf("expected 422 or 400, got %d", w.Code)
		}
	})

	t.Run("short password", func(t *testing.T) {
		body := `{"email":"test@example.com","username":"user","password":"ab"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
			t.Errorf("expected 422 or 400, got %d", w.Code)
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		body := `{"email":"not-an-email","username":"user","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
			t.Errorf("expected 422 or 400, got %d", w.Code)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 400 or 422, got %d", w.Code)
		}
	})
}

func TestAdminSetUserRole_Success(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "role@example.com", "roleuser", "pass")

	router := newTestEngine()
	router.PATCH("/admin/users/:id/role", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminSetUserRole(svc))

	body := `{"role":"admin"}`
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/role", u.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAdminSetUserRole_InvalidRole(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "role@example.com", "roleuser", "pass")

	router := newTestEngine()
	router.PATCH("/admin/users/:id/role", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminSetUserRole(svc))

	body := `{"role":"superadmin"}`
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/role", u.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Errorf("expected 422 or 400, got %d", w.Code)
	}
}

func TestAdminSetUserRole_InvalidID(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.PATCH("/admin/users/:id/role", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminSetUserRole(svc))

	body := `{"role":"admin"}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/abc/role", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminResetUserPassword_Success(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "pw@example.com", "pwuser", "oldpass")

	router := newTestEngine()
	router.POST("/admin/users/:id/password", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminResetUserPassword(svc))

	body := `{"password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%d/password", u.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAdminResetUserPassword_ShortPassword(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "pw@example.com", "pwuser", "oldpass")

	router := newTestEngine()
	router.POST("/admin/users/:id/password", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminResetUserPassword(svc))

	body := `{"password":"ab"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%d/password", u.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Errorf("expected 422 or 400, got %d", w.Code)
	}
}

func TestAdminSetUserActive_Success(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "active@example.com", "activeuser", "pass")

	router := newTestEngine()
	router.PATCH("/admin/users/:id/active", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminSetUserActive(svc))

	body := `{"active":false}`
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/active", u.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestAdminDeleteUser_Success(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "delete@example.com", "deleteuser", "pass")

	router := newTestEngine()
	router.DELETE("/admin/users/:id", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminDeleteUser(svc))

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%d", u.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["deleted"].(float64) != 1 {
		t.Errorf("expected deleted=1, got %v", resp["deleted"])
	}
}

func TestAdminDeleteUser_InvalidID(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.DELETE("/admin/users/:id", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminDeleteUser(svc))

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminCreateChannel_Success(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	u, _ := userRepo.Create(t.Context(), "ch@example.com", "chuser", "pass")

	router := newTestEngine()
	router.POST("/admin/channels", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminCreateChannel(svc))

	body := fmt.Sprintf(`{"user_id":%d,"kind":"feishu","name":"Test Channel","configuration":{"webhook_url":"https://example.com","card_link_url":""}}`, u.ID)
	req := httptest.NewRequest(http.MethodPost, "/admin/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d, body: %s", http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestAdminCreateChannel_InvalidBody(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.POST("/admin/channels", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminCreateChannel(svc))

	t.Run("missing required fields", func(t *testing.T) {
		body := `{"name":"Test"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
			t.Errorf("expected 422 or 400, got %d", w.Code)
		}
	})

	t.Run("invalid configuration JSON", func(t *testing.T) {
		body := `{"user_id":1,"kind":"feishu","name":"Test","configuration":"not-json"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 400 or 422, got %d", w.Code)
		}
	})
}

func TestAdminGetStats_Success(t *testing.T) {
	// AdminGetStats must report the real totals (the COUNT(*) from each
	// repository), not len() of a limit=1 slice. Build the service in-test so we
	// keep the db handle and can seed known counts for every entity.
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	postRepo := infra.NewPostRepository(db)
	channelRepo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)
	svc := admin.NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
		&mockSessionLister{},
		&mockAuditRecorder{},
	)

	ctx := t.Context()
	u1, _ := userRepo.Create(ctx, "a@example.com", "alice", "pass")
	u2, _ := userRepo.Create(ctx, "b@example.com", "bob", "pass")
	for i := 0; i < 3; i++ {
		_, _ = postRepo.Create(ctx, fmt.Sprintf("post-%d", i), "body", u1.ID)
	}
	_ = channelRepo.Create(ctx, &delivery.Channel{
		UserID:        u2.ID,
		Kind:          delivery.ChannelKindFeishu,
		Name:          "Ch1",
		Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""},
	})
	// delivery_history has no repository Create; seed via raw SQL (existing
	// convention in delivery_attempt_repo_test.go).
	for i := 0; i < 4; i++ {
		if err := db.Exec(
			"INSERT INTO delivery_history (status, last_error, user_id) VALUES (?, '', ?)",
			delivery.StatusDelivered, u1.ID,
		).Error; err != nil {
			t.Fatalf("seed delivery_history: %v", err)
		}
	}

	router := newTestEngine()
	router.GET("/admin/stats", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminGetStats(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	counts, ok := resp["counts"].(map[string]interface{})
	if !ok {
		t.Fatal("expected counts in response")
	}
	want := map[string]float64{
		"users":    2,
		"posts":    3,
		"channels": 1,
		"history":  4,
	}
	for key, expected := range want {
		got, ok := counts[key].(float64)
		if !ok {
			t.Errorf("expected %s count to be numeric, got %T (%v)", key, counts[key], counts[key])
			continue
		}
		if got != expected {
			t.Errorf("expected %s count %v, got %v", key, expected, got)
		}
	}
}

func TestAdminListSessions_Success(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.GET("/admin/users/:id/sessions", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListSessions(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/users/1/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["sessions"]; !ok {
		t.Error("expected sessions in response")
	}
}

func TestAdminListSessions_InvalidID(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.GET("/admin/users/:id/sessions", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListSessions(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/users/abc/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminRevokeUserSessions_Success(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.DELETE("/admin/users/:id/sessions", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminRevokeUserSessions(svc))

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/1/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["revoked"] != true {
		t.Errorf("expected revoked=true, got %v", resp["revoked"])
	}
}

func TestAdminRevokeUserSessions_InvalidID(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.DELETE("/admin/users/:id/sessions", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminRevokeUserSessions(svc))

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/abc/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAdminListAuditLogs_Success(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.GET("/admin/audit-logs", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListAuditLogs(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["items"]; !ok {
		t.Error("expected items in response")
	}
	if _, ok := resp["total"]; !ok {
		t.Error("expected total in response")
	}
}

func TestAdminListDeliveryHistory_Success(t *testing.T) {
	svc, _, _ := setupAdminHandlerWithMutators(t)

	router := newTestEngine()
	router.GET("/admin/delivery-history", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Role: user.RoleAdmin})
		c.Next()
	}, AdminListDeliveryHistory(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/delivery-history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["items"]; !ok {
		t.Error("expected items in response")
	}
}
