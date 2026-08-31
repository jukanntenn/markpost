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

// retentionRouter registers one admin retention route with an admin actor.
func retentionRouter(t *testing.T, method, path string, actorID int, handler gin.HandlerFunc) *gin.Engine {
	t.Helper()
	router := newTestEngine()
	router.Handle(method, path, func(c *gin.Context) {
		c.Set("user", &user.User{ID: actorID, Role: user.RoleAdmin})
		c.Next()
	}, handler)
	return router
}

func TestAdminSetUserRetention(t *testing.T) {
	t.Run("sets, echoes, and clears the policy", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@ret.com", "actorret", "pass")
		u, _ := userRepo.Create(t.Context(), "target@ret.com", "targetret", "pass")

		router := retentionRouter(t, http.MethodPatch, "/admin/users/:id/retention", actor.ID, AdminSetUserRetention(svc))

		steps := []struct {
			body string
			want string
		}{
			{`{"retention_days":30}`, `"retention_days":30`},
			{`{"retention_days":0}`, `"retention_days":0`},
			{`{"retention_days":null}`, `"retention_days":null`},
		}
		for _, st := range steps {
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/retention", u.ID), strings.NewReader(st.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), st.want) {
				t.Errorf("expected %s in body: %s", st.want, w.Body.String())
			}
		}
	})

	t.Run("rejects out-of-range days", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@ret2.com", "actorret2", "pass")
		u, _ := userRepo.Create(t.Context(), "target@ret2.com", "targetret2", "pass")

		router := retentionRouter(t, http.MethodPatch, "/admin/users/:id/retention", actor.ID, AdminSetUserRetention(svc))

		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/retention", u.ID), strings.NewReader(`{"retention_days":4000}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 for a nonexistent user", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@ret3.com", "actorret3", "pass")

		router := retentionRouter(t, http.MethodPatch, "/admin/users/:id/retention", actor.ID, AdminSetUserRetention(svc))

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/99999/retention", strings.NewReader(`{"retention_days":7}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestAdminBulkSetRetention(t *testing.T) {
	t.Run("writes explicit ids and reports the count", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@bulk.com", "actorbulk", "pass")
		a, _ := userRepo.Create(t.Context(), "a@bulk.com", "abulk", "pass")
		b, _ := userRepo.Create(t.Context(), "b@bulk.com", "bbulk", "pass")

		router := retentionRouter(t, http.MethodPost, "/admin/users/retention/bulk", actor.ID, AdminBulkSetRetention(svc))

		body := fmt.Sprintf(`{"user_ids":[%d,%d],"retention_days":90}`, a.ID, b.ID)
		req := httptest.NewRequest(http.MethodPost, "/admin/users/retention/bulk", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"updated":2`) {
			t.Errorf("expected updated=2, body: %s", w.Body.String())
		}
		for _, uid := range []int{a.ID, b.ID} {
			got, err := userRepo.GetByID(t.Context(), uid)
			if err != nil || got.RetentionDays == nil || *got.RetentionDays != 90 {
				t.Errorf("user %d retention = %v, want 90", uid, got.RetentionDays)
			}
		}
	})

	t.Run("scope=vip writes only VIP users", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@bulk2.com", "actorbulk2", "pass")
		vipUser, _ := userRepo.Create(t.Context(), "vip@bulk.com", "vipbulk", "pass")
		plain, _ := userRepo.Create(t.Context(), "plain@bulk.com", "plainbulk", "pass")

		if err := userRepo.SetUserVIP(t.Context(), vipUser.ID, true, nil); err != nil {
			t.Fatalf("grant vip: %v", err)
		}

		router := retentionRouter(t, http.MethodPost, "/admin/users/retention/bulk", actor.ID, AdminBulkSetRetention(svc))

		req := httptest.NewRequest(http.MethodPost, "/admin/users/retention/bulk", strings.NewReader(`{"scope":"vip","retention_days":0}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}
		got, _ := userRepo.GetByID(t.Context(), vipUser.ID)
		if got.RetentionDays == nil || *got.RetentionDays != 0 {
			t.Errorf("vip user retention = %v, want 0 (forever)", got.RetentionDays)
		}
		gotPlain, _ := userRepo.GetByID(t.Context(), plain.ID)
		if gotPlain.RetentionDays != nil {
			t.Errorf("plain user retention = %v, want nil", gotPlain.RetentionDays)
		}
	})

	t.Run("requires a target", func(t *testing.T) {
		svc, userRepo, _ := setupAdminHandlerWithMutators(t)
		actor, _ := userRepo.Create(t.Context(), "actor@bulk3.com", "actorbulk3", "pass")

		router := retentionRouter(t, http.MethodPost, "/admin/users/retention/bulk", actor.ID, AdminBulkSetRetention(svc))

		req := httptest.NewRequest(http.MethodPost, "/admin/users/retention/bulk", strings.NewReader(`{"retention_days":7}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d, body: %s", w.Code, w.Body.String())
		}
	})
}

func TestAdminRetentionImpact(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	actor, _ := userRepo.Create(t.Context(), "actor@imp.com", "actorimp", "pass")
	a, _ := userRepo.Create(t.Context(), "a@imp.com", "aimp", "pass")

	router := retentionRouter(t, http.MethodPost, "/admin/retention/impact", actor.ID, AdminRetentionImpact(svc))

	body := fmt.Sprintf(`{"user_ids":[%d],"retention_days":0}`, a.ID)
	req := httptest.NewRequest(http.MethodPost, "/admin/retention/impact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	// A forever candidate matches nothing regardless of row age.
	for _, want := range []string{`"users_affected":1`, `"posts_to_delete":0`, `"history_to_delete":0`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("expected %s in body: %s", want, w.Body.String())
		}
	}
}

func TestAdminRetentionDefaults(t *testing.T) {
	svc, userRepo, _ := setupAdminHandlerWithMutators(t)
	actor, _ := userRepo.Create(t.Context(), "actor@def.com", "actordef", "pass")

	router := retentionRouter(t, http.MethodGet, "/admin/retention/defaults", actor.ID, AdminRetentionDefaults(svc))

	req := httptest.NewRequest(http.MethodGet, "/admin/retention/defaults", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"post_retention_days":7`) {
		t.Errorf("expected the mirrored global default 7, body: %s", w.Body.String())
	}
}
