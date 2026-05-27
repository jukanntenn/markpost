package user

import "testing"

func TestIsAdmin(t *testing.T) {
	t.Run("admin user", func(t *testing.T) {
		u := User{Role: RoleAdmin}
		if !u.IsAdmin() {
			t.Error("expected IsAdmin() to return true for admin role")
		}
	})

	t.Run("regular user", func(t *testing.T) {
		u := User{Role: RoleUser}
		if u.IsAdmin() {
			t.Error("expected IsAdmin() to return false for user role")
		}
	})
}
