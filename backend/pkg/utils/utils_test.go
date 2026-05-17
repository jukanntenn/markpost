package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
)

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func TestHashToken(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		input := ""
		got := HashToken(input)
		if got != sha256Hex(input) {
			t.Errorf("HashToken(%q) = %q, want %q", input, got, sha256Hex(input))
		}
	})

	t.Run("simple string", func(t *testing.T) {
		input := "hello"
		got := HashToken(input)
		if got != sha256Hex(input) {
			t.Errorf("HashToken(%q) = %q, want %q", input, got, sha256Hex(input))
		}
	})

	t.Run("long token", func(t *testing.T) {
		input := strings.Repeat("a", 64)
		got := HashToken(input)
		if got != sha256Hex(input) {
			t.Errorf("HashToken(64-byte string) = %q, want %q", got, sha256Hex(input))
		}
	})

	t.Run("returns 64 hex characters", func(t *testing.T) {
		got := HashToken("test")
		if len(got) != 64 {
			t.Errorf("HashToken result length = %d, want 64", len(got))
		}
		matched, _ := regexp.MatchString("^[0-9a-f]+$", got)
		if !matched {
			t.Errorf("HashToken result %q is not lowercase hex", got)
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		a := HashToken("consistent")
		b := HashToken("consistent")
		if a != b {
			t.Errorf("HashToken is not deterministic: %q != %q", a, b)
		}
	})
}

func TestGeneratePostKey(t *testing.T) {
	alnum := regexp.MustCompile("^[A-Za-z0-9]+$")

	t.Run("default length", func(t *testing.T) {
		key, err := GeneratePostKey(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(key, "mpk-") {
			t.Errorf("key %q should have prefix mpk-", key)
		}
		if len(key) != 24 {
			t.Errorf("key length = %d, want 24 (4 prefix + 20 random)", len(key))
		}
		random := strings.TrimPrefix(key, "mpk-")
		if !alnum.MatchString(random) {
			t.Errorf("random portion %q should be alphanumeric", random)
		}
	})

	t.Run("custom length", func(t *testing.T) {
		key, err := GeneratePostKey(15)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(key, "mpk-") {
			t.Errorf("key %q should have prefix mpk-", key)
		}
		if len(key) != 19 {
			t.Errorf("key length = %d, want 19 (4 prefix + 15 random)", len(key))
		}
		random := strings.TrimPrefix(key, "mpk-")
		if !alnum.MatchString(random) {
			t.Errorf("random portion %q should be alphanumeric", random)
		}
	})

	t.Run("length of 1", func(t *testing.T) {
		key, err := GeneratePostKey(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != 5 {
			t.Errorf("key length = %d, want 5 (4 prefix + 1 random)", len(key))
		}
	})

	t.Run("negative length uses default", func(t *testing.T) {
		key, err := GeneratePostKey(-5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != 24 {
			t.Errorf("key length = %d, want 24 for negative input", len(key))
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		a, _ := GeneratePostKey(0)
		b, _ := GeneratePostKey(0)
		if a == b {
			t.Errorf("two generated keys should differ: %q == %q", a, b)
		}
	})
}

func TestGenerateRandomPassword(t *testing.T) {
	validChars := regexp.MustCompile("^[a-zA-Z0-9!@#$%^&*]+$")

	t.Run("default length", func(t *testing.T) {
		pw, err := GenerateRandomPassword(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pw) != 12 {
			t.Errorf("password length = %d, want 12", len(pw))
		}
		if !validChars.MatchString(pw) {
			t.Errorf("password %q contains invalid characters", pw)
		}
	})

	t.Run("custom length", func(t *testing.T) {
		pw, err := GenerateRandomPassword(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pw) != 20 {
			t.Errorf("password length = %d, want 20", len(pw))
		}
		if !validChars.MatchString(pw) {
			t.Errorf("password %q contains invalid characters", pw)
		}
	})

	t.Run("negative length uses default", func(t *testing.T) {
		pw, err := GenerateRandomPassword(-1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pw) != 12 {
			t.Errorf("password length = %d, want 12 for negative input", len(pw))
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		a, _ := GenerateRandomPassword(0)
		b, _ := GenerateRandomPassword(0)
		if a == b {
			t.Errorf("two generated passwords should differ: %q == %q", a, b)
		}
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("hashes password", func(t *testing.T) {
		hash, err := HashPassword("secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash == "" {
			t.Error("expected non-empty hash")
		}
		if hash == "secret123" {
			t.Error("hash should not equal plain password")
		}
	})

	t.Run("different passwords produce different hashes", func(t *testing.T) {
		a, _ := HashPassword("password1")
		b, _ := HashPassword("password2")
		if a == b {
			t.Error("different passwords should produce different hashes")
		}
	})

	t.Run("same password produces different hashes (bcrypt salt)", func(t *testing.T) {
		a, _ := HashPassword("samepassword")
		b, _ := HashPassword("samepassword")
		if a == b {
			t.Error("bcrypt should produce different hashes for same password due to salt")
		}
	})
}

func TestCheckPassword(t *testing.T) {
	t.Run("correct password", func(t *testing.T) {
		hash, _ := HashPassword("mypassword")
		ok, err := CheckPassword("mypassword", hash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected password to match")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		hash, _ := HashPassword("mypassword")
		ok, err := CheckPassword("wrongpassword", hash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected password not to match")
		}
	})

	t.Run("invalid hash", func(t *testing.T) {
		_, err := CheckPassword("test", "not-a-valid-hash")
		if err == nil {
			t.Error("expected error for invalid hash")
		}
	})
}

func TestGenerateState(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		state, err := GenerateState()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state == "" {
			t.Error("expected non-empty state")
		}
	})

	t.Run("is base64url encoded", func(t *testing.T) {
		state, err := GenerateState()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		matched, _ := regexp.MatchString("^[A-Za-z0-9_-]+$", state)
		if !matched {
			t.Errorf("state %q should be base64url encoded (no padding)", state)
		}
	})

	t.Run("correct length for 20 bytes", func(t *testing.T) {
		state, err := GenerateState()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(state) != 27 {
			t.Errorf("state length = %d, want 27 (base64 of 20 bytes, no padding)", len(state))
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		a, _ := GenerateState()
		b, _ := GenerateState()
		if a == b {
			t.Errorf("two generated states should differ: %q == %q", a, b)
		}
	})
}
