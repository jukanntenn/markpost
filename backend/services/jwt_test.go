package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testAccessSigningKey  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testRefreshSigningKey = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func newTestJWTService() *JWTService {
	return NewJWTService(testAccessSigningKey, testRefreshSigningKey, time.Hour, 24*time.Hour)
}

func TestJWTService_TokensAndValidation(t *testing.T) {
	jwtSvc := newTestJWTService()

	t.Run("access token success", func(t *testing.T) {
		token, err := jwtSvc.GenerateAccessToken(42, "user")
		if err != nil || token == "" {
			t.Fatalf("GenerateAccessToken error: %v", err)
		}
		claims, err := jwtSvc.ValidateAccess(token)
		if err != nil || claims == nil {
			t.Fatalf("ValidateAccess error: %v", err)
		}
		if claims.Subject != "urn:user:42" {
			t.Fatalf("unexpected subject: %s", claims.Subject)
		}
		id, err := claims.UserID()
		if err != nil || id != 42 {
			t.Fatalf("GetUserIDFromClaims unexpected: %d %v", id, err)
		}
	})

	t.Run("access validate fails on refresh token", func(t *testing.T) {
		token, _ := jwtSvc.GenerateRefreshToken(7, "user")
		if _, err := jwtSvc.ValidateAccess(token); err == nil {
			t.Fatalf("expected error when validating refresh token as access")
		}
	})

	t.Run("invalid issuer rejected", func(t *testing.T) {
		now := time.Now()
		claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "urn:user:1",
			Issuer:    "other",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		}}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		s, _ := tok.SignedString([]byte(testAccessSigningKey))
		if _, err := jwtSvc.ValidateAccess(s); err == nil {
			t.Fatalf("expected issuer error")
		}
	})

	t.Run("invalid subject numeric conversion", func(t *testing.T) {
		now := time.Now()
		claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "urn:user:abc",
			Issuer:    "markpost",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		}}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		s, _ := tok.SignedString([]byte(testAccessSigningKey))
		c, err := jwtSvc.ValidateAccess(s)
		if err != nil || c == nil {
			t.Fatalf("unexpected validate error: %v", err)
		}
		if _, err := c.UserID(); err == nil {
			t.Fatalf("expected conversion error")
		}
	})

	t.Run("pair generation and both validate", func(t *testing.T) {
		pair, err := jwtSvc.GenerateTokenPair(88, "user")
		if err != nil || pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatalf("GenerateTokenPair error: %v", err)
		}
		if _, err := jwtSvc.ValidateAccess(pair.AccessToken); err != nil {
			t.Fatalf("ValidateAccess error: %v", err)
		}
		if _, err := jwtSvc.ValidateRefresh(pair.RefreshToken); err != nil {
			t.Fatalf("ValidateRefresh error: %v", err)
		}
	})
}
