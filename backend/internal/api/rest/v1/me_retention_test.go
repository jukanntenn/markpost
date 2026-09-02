package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/service/me"

	"github.com/gin-gonic/gin"
)

// meRetentionRouter registers GET /me/retention behind a caller-injecting
// middleware, mirroring how AuthWithBlacklist seeds the context user.
func meRetentionRouter(u *user.User, svc *me.Service) *gin.Engine {
	router := newTestEngine()
	router.GET("/me/retention", func(c *gin.Context) {
		if u != nil {
			c.Set("user", u)
		}
		c.Next()
	}, MeRetention(svc))
	return router
}

func TestMeRetention(t *testing.T) {
	forever, thirty := 0, 30
	svc := me.NewService(7, 168*time.Hour)

	cases := []struct {
		name string
		u    *user.User
		want string
	}{
		{"explicit forever", &user.User{RetentionDays: &forever}, `{"posts_days":0,"history_days":0}`},
		{"explicit N days", &user.User{RetentionDays: &thirty}, `{"posts_days":30,"history_days":30}`},
		{"inherit resolves globals", &user.User{}, `{"posts_days":7,"history_days":7}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := meRetentionRouter(tc.u, svc)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me/retention", nil))

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.want) {
				t.Errorf("expected %s in body: %s", tc.want, w.Body.String())
			}
		})
	}
}

// The unauthenticated 401 belongs to AuthWithBlacklist (covered by the
// middleware suite); without a context user the handler only fails closed.
func TestMeRetentionWithoutUser(t *testing.T) {
	router := meRetentionRouter(nil, me.NewService(7, 168*time.Hour))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me/retention", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body: %s", w.Code, w.Body.String())
	}
}
