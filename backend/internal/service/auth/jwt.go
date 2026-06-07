// Package auth provides JWT token generation and validation services.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTTokenPair contains an access token and refresh token pair.
type JWTTokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// ExpiresInSeconds returns the number of seconds until the token pair expires.
func (t *JWTTokenPair) ExpiresInSeconds() int64 {
	return int64(time.Until(t.ExpiresAt).Seconds())
}

// AccessClaims contains the claims embedded in access tokens.
type AccessClaims struct {
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// RefreshClaims contains the claims embedded in refresh tokens.
type RefreshClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token generation and validation.
type JWTService struct {
	accessSigningKey   []byte
	refreshSigningKey  []byte
	accessTokenExpire  time.Duration
	refreshTokenExpire time.Duration
}

// NewJWTService creates a new JWTService with the provided signing keys and expiration durations.
func NewJWTService(accessSigningKey, refreshSigningKey string, accessTokenExpire, refreshTokenExpire time.Duration) *JWTService {
	return &JWTService{
		accessSigningKey:   []byte(accessSigningKey),
		refreshSigningKey:  []byte(refreshSigningKey),
		accessTokenExpire:  accessTokenExpire,
		refreshTokenExpire: refreshTokenExpire,
	}
}

// GenerateTokenPair generates a new access and refresh token pair for the user.
func (s *JWTService) GenerateTokenPair(userID int, email, username, role string) (*JWTTokenPair, error) {
	now := time.Now()
	accessToken, err := s.GenerateAccessToken(now, userID, email, username, role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(now, userID, role)
	if err != nil {
		return nil, err
	}

	return &JWTTokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    now.Add(s.accessTokenExpire),
	}, nil
}

// GenerateAccessToken generates a new access token for the user.
func (s *JWTService) GenerateAccessToken(now time.Time, userID int, email, username, role string) (string, error) {
	expiresAt := now.Add(s.accessTokenExpire)
	claims := AccessClaims{
		UserID:   userID,
		Email:    email,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.accessSigningKey)
}

// GenerateRefreshToken generates a new refresh token for the user.
func (s *JWTService) GenerateRefreshToken(now time.Time, userID int, role string) (string, error) {
	claims := RefreshClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.refreshSigningKey)
}

// ValidateAccess validates an access token and returns its claims.
func (s *JWTService) ValidateAccess(tokenString string) (*AccessClaims, error) {
	return validateTokenClaims(tokenString, s.accessSigningKey, func() *AccessClaims { return &AccessClaims{} })
}

// ValidateRefresh validates a refresh token and returns its claims.
func (s *JWTService) ValidateRefresh(tokenString string) (*RefreshClaims, error) {
	return validateTokenClaims(tokenString, s.refreshSigningKey, func() *RefreshClaims { return &RefreshClaims{} })
}

func validateTokenClaims[T jwt.Claims](tokenString string, key []byte, newClaims func() T) (T, error) {
	var zero T
	claims, err := validateToken(tokenString, key, func() jwt.Claims { return newClaims() })
	if err != nil {
		return zero, err
	}
	typed, ok := claims.(T)
	if !ok {
		return zero, jwt.ErrSignatureInvalid
	}
	return typed, nil
}

func validateToken(tokenString string, key []byte, newClaims func() jwt.Claims) (jwt.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, newClaims(), func(_ *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if token.Valid {
		return token.Claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}
