package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"markpost/internal/domain/settings"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
	adminsvc "markpost/internal/service/admin"

	"github.com/gin-gonic/gin"
)

func setupAdminSettings(t *testing.T) (*adminsvc.Service, settings.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	settingsRepo := infra.NewSettingsRepository(db)
	concreteUserRepo, ok := infra.NewUserRepository(db, 16).(*infra.UserRepository)
	if !ok {
		t.Fatal("NewUserRepository did not return *infra.UserRepository")
	}
	svc := adminsvc.NewService(
		concreteUserRepo,
		&postListerAdapter{repo: infra.NewPostRepository(db)},
		&channelListerAdapter{repo: infra.NewDeliveryChannelRepository(db)},
		infra.NewAttemptRepository(db),
		&mockSessionLister{},
		&mockAuditRecorder{},
	)
	svc.SetSettingsStore(settingsRepo)
	return svc, settingsRepo
}

func TestAdminSettings(t *testing.T) {
	actor := &user.User{ID: 1, Role: user.RoleAdmin}

	t.Run("lists the stored vip setting", func(t *testing.T) {
		svc, setting := setupAdminSettings(t)
		if err := setting.Set(t.Context(), settings.KeyVIP, settings.SettingValue{Enabled: true}, 0); err != nil {
			t.Fatalf("seed vip strategy: %v", err)
		}
		router := newTestEngine()
		router.GET("/admin/settings", func(c *gin.Context) { c.Set("user", actor); c.Next() }, AdminGetSettings(svc))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/settings", nil))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"key":"vip"`) || !strings.Contains(w.Body.String(), `"enabled":true`) {
			t.Errorf("expected stored vip setting in body: %s", w.Body.String())
		}
	})

	t.Run("upserts the vip setting", func(t *testing.T) {
		svc, _ := setupAdminSettings(t)
		router := newTestEngine()
		router.PUT("/admin/settings/:key", func(c *gin.Context) { c.Set("user", actor); c.Next() }, AdminSetSetting(svc))

		req := httptest.NewRequest(http.MethodPut, "/admin/settings/vip", strings.NewReader(`{"enabled":false}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"enabled":false`) {
			t.Errorf("expected disabled vip in body: %s", w.Body.String())
		}
	})

	t.Run("rejects an unknown key", func(t *testing.T) {
		svc, _ := setupAdminSettings(t)
		router := newTestEngine()
		router.PUT("/admin/settings/:key", func(c *gin.Context) { c.Set("user", actor); c.Next() }, AdminSetSetting(svc))

		req := httptest.NewRequest(http.MethodPut, "/admin/settings/other", strings.NewReader(`{"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d, body: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "unknown_setting") {
			t.Errorf("expected unknown_setting error code, body: %s", w.Body.String())
		}
	})
}
