package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

	svc := admin.NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
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
	json.Unmarshal(w.Body.Bytes(), &resp)
	users, ok := resp["users"].([]interface{})
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
	json.Unmarshal(w.Body.Bytes(), &resp)
	pagination, ok := resp["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("expected pagination in response")
	}
	if int(pagination["total"].(float64)) != 3 {
		t.Errorf("expected total 3, got %v", pagination["total"])
	}
}

func TestAdminListPosts_Success(t *testing.T) {
	svc, _, postRepo, _ := setupAdminHandlerDeps(t)
	ctx := t.Context()
	_, _ = postRepo.Create(ctx, "Post 1", "Body 1", 1)
	_, _ = postRepo.Create(ctx, "Post 2", "Body 2", 2)

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
	json.Unmarshal(w.Body.Bytes(), &resp)
	posts, ok := resp["posts"].([]interface{})
	if !ok {
		t.Fatal("expected posts array in response")
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}
}

func TestAdminListPosts_WithSearch(t *testing.T) {
	svc, _, postRepo, _ := setupAdminHandlerDeps(t)
	ctx := t.Context()
	_, _ = postRepo.Create(ctx, "Alert Post", "Body", 1)
	_, _ = postRepo.Create(ctx, "Normal Post", "Body", 2)

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
	json.Unmarshal(w.Body.Bytes(), &resp)
	posts, _ := resp["posts"].([]interface{})
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
}

func TestAdminListChannels_Success(t *testing.T) {
	svc, _, _, channelRepo := setupAdminHandlerDeps(t)
	ctx := t.Context()
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", WebhookURL: "https://a.com"})
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: 2, Kind: delivery.ChannelKindFeishu, Name: "Ch2", WebhookURL: "https://b.com"})

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
	json.Unmarshal(w.Body.Bytes(), &resp)
	channels, ok := resp["channels"].([]interface{})
	if !ok {
		t.Fatal("expected channels array in response")
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}
