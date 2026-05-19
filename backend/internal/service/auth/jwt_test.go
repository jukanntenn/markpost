package auth

import (
	"testing"
	"time"
)

func newTestJWTService() *JWTService {
	return NewJWTService(
		"test-access-signing-key-min-32-characters!!",
		"test-refresh-signing-key-min-32-characters!!",
		time.Hour,
		time.Hour*24*7,
	)
}

func TestJWTService_GenerateTokenPair(t *testing.T) {
	jwtSvc := newTestJWTService()

	pair, err := jwtSvc.GenerateTokenPair(1, "test@example.com", "testuser", "user")
	if err != nil {
		t.Fatalf("GenerateTokenPair error: %v", err)
	}

	if pair.AccessToken == "" {
		t.Fatal("AccessToken is empty")
	}

	if pair.RefreshToken == "" {
		t.Fatal("RefreshToken is empty")
	}

	if pair.ExpiresAt.IsZero() {
		t.Fatal("ExpiresAt is zero")
	}
}

func TestJWTService_ValidateAccess(t *testing.T) {
	jwtSvc := newTestJWTService()

	token, err := jwtSvc.GenerateAccessToken(time.Now(), 1, "test@example.com", "testuser", "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	claims, err := jwtSvc.ValidateAccess(token)
	if err != nil {
		t.Fatalf("ValidateAccess error: %v", err)
	}

	if claims.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", claims.UserID)
	}

	if claims.Email != "test@example.com" {
		t.Fatalf("expected Email 'test@example.com', got %s", claims.Email)
	}

	if claims.Username != "testuser" {
		t.Fatalf("expected Username 'testuser', got %s", claims.Username)
	}

	if claims.Role != "user" {
		t.Fatalf("expected Role 'user', got %s", claims.Role)
	}
}

func TestJWTService_ValidateRefresh(t *testing.T) {
	jwtSvc := newTestJWTService()

	token, err := jwtSvc.GenerateRefreshToken(time.Now(), 1, "user")
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}

	claims, err := jwtSvc.ValidateRefresh(token)
	if err != nil {
		t.Fatalf("ValidateRefresh error: %v", err)
	}

	if claims.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", claims.UserID)
	}
}

func TestJWTService_ValidateAccess_InvalidToken(t *testing.T) {
	jwtSvc := newTestJWTService()

	_, err := jwtSvc.ValidateAccess("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestJWTService_ValidateRefresh_InvalidToken(t *testing.T) {
	jwtSvc := newTestJWTService()

	_, err := jwtSvc.ValidateRefresh("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
