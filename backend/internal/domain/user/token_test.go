package user

import (
	"testing"
	"time"
)

func TestRefreshToken_IsExpired(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		token := RefreshToken{ExpiresAt: time.Now().Add(-time.Hour)}
		if !token.IsExpired() {
			t.Error("expected token to be expired")
		}
	})

	t.Run("not expired", func(t *testing.T) {
		token := RefreshToken{ExpiresAt: time.Now().Add(time.Hour)}
		if token.IsExpired() {
			t.Error("expected token to not be expired")
		}
	})
}

func TestRefreshToken_TableName(t *testing.T) {
	token := RefreshToken{}
	if token.TableName() != "refresh_tokens" {
		t.Errorf("TableName() = %q, want %q", token.TableName(), "refresh_tokens")
	}
}

func TestTokenBlacklist_TableName(t *testing.T) {
	bl := TokenBlacklist{}
	if bl.TableName() != "token_blacklist" {
		t.Errorf("TableName() = %q, want %q", bl.TableName(), "token_blacklist")
	}
}
