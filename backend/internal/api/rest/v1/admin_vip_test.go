package v1

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"markpost/internal/domain/user"

	"github.com/gin-gonic/gin"
)

func TestAdminSetUserVIP(t *testing.T) {
	t.Run("grants and revokes vip", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@vip.com", "actorvip", "pass")
		u, _ := userRepo.Create(t.Context(), "vip-target@example.com", "viptarget", "pass")

		router := newTestEngine()
		router.PATCH("/admin/users/:id/vip", func(c *gin.Context) {
			c.Set("user", &user.User{ID: actor.ID, Role: user.RoleAdmin})
			c.Next()
		}, AdminSetUserVIP(svc))

		for _, tc := range []struct {
			body string
			want bool
		}{
			{`{"vip":true}`, true},
			{`{"vip":false}`, false},
		} {
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/vip", u.ID), strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
			}
			want := fmt.Sprintf(`"vip":%v`, tc.want)
			if !strings.Contains(w.Body.String(), want) {
				t.Errorf("expected %s in body: %s", want, w.Body.String())
			}
		}
	})

	t.Run("returns 404 for a nonexistent user", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@vip404.com", "actorvip404", "pass")

		router := newTestEngine()
		router.PATCH("/admin/users/:id/vip", func(c *gin.Context) {
			c.Set("user", &user.User{ID: actor.ID, Role: user.RoleAdmin})
			c.Next()
		}, AdminSetUserVIP(svc))

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/99999/vip", strings.NewReader(`{"vip":true}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}
