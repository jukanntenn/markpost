// Package auth provides authentication services including OAuth, JWT token management,
// and user session handling.
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
	expiresAt := time.Now().Add(s.accessTokenExpire)
	accessToken, err := s.GenerateAccessToken(userID, email, username, role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(userID, role)
	if err != nil {
		return nil, err
	}

	return &JWTTokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// GenerateAccessToken generates a new access token for the user.
func (s *JWTService) GenerateAccessToken(userID int, email, username, role string) (string, error) {
	now := time.Now()
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
func (s *JWTService) GenerateRefreshToken(userID int, role string) (string, error) {
	claims := RefreshClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.refreshSigningKey)
}

// ValidateAccess validates an access token and returns its claims.
func (s *JWTService) ValidateAccess(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return s.accessSigningKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AccessClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// ValidateRefresh validates a refresh token and returns its claims.
func (s *JWTService) ValidateRefresh(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return s.refreshSigningKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
