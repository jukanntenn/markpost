package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"markpost/internal/domain/user"

	"github.com/gin-gonic/gin"
)

// meProfileRouter registers GET /me behind a caller-injecting middleware,
// mirroring how AuthWithBlacklist seeds the context user.
func meProfileRouter(u *user.User) *gin.Engine {
	router := newTestEngine()
	router.GET("/me", func(c *gin.Context) {
		if u != nil {
			c.Set("user", u)
		}
		c.Next()
	}, MeProfile())
	return router
}

// The handler serializes the context user as-is (the middleware already
// re-read the row), so the suite only pins the field surface — vip included,
// which is the field that motivated the endpoint.
func TestMeProfile(t *testing.T) {
	u := &user.User{
		ID:       7,
		Email:    "u@example.com",
		Username: "u",
		Name:     "U",
		Role:     user.RoleUser,
		VIP:      true,
	}

	router := meProfileRouter(u)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	for _, want := range []string{`"id":7`, `"username":"u"`, `"vip":true`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("expected %s in body: %s", want, w.Body.String())
		}
	}
}

// The unauthenticated 401 belongs to AuthWithBlacklist (covered by the
// middleware suite); without a context user the handler only fails closed.
func TestMeProfileWithoutUser(t *testing.T) {
	router := meProfileRouter(nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body: %s", w.Code, w.Body.String())
	}
}
