// Package auth provides JWT authentication utilities.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTTokenPair represents a pair of access and refresh tokens.
type JWTTokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AccessClaims represents JWT access token claims.
type AccessClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// RefreshClaims represents JWT refresh token claims.
type RefreshClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService provides JWT token generation and validation.
type JWTService struct {
	accessSigningKey   []byte
	refreshSigningKey  []byte
	accessTokenExpire  time.Duration
	refreshTokenExpire time.Duration
}

// NewJWTService creates a new JWTService instance.
func NewJWTService(accessSigningKey, refreshSigningKey string, accessTokenExpire, refreshTokenExpire time.Duration) *JWTService {
	return &JWTService{
		accessSigningKey:   []byte(accessSigningKey),
		refreshSigningKey:  []byte(refreshSigningKey),
		accessTokenExpire:  accessTokenExpire,
		refreshTokenExpire: refreshTokenExpire,
	}
}

// GenerateTokenPair generates a new pair of access and refresh tokens.
func (s *JWTService) GenerateTokenPair(userID int, role string) (*JWTTokenPair, error) {
	accessToken, err := s.GenerateAccessToken(userID, role)
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
	}, nil
}

// GenerateAccessToken generates a new access token.
func (s *JWTService) GenerateAccessToken(userID int, role string) (string, error) {
	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.accessSigningKey)
}

// GenerateRefreshToken generates a new refresh token.
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

// ValidateAccess validates an access token.
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

// ValidateRefresh validates a refresh token.
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

// UserIDInt returns the user ID as an int.
func (c *AccessClaims) UserIDInt() int {
	return c.UserID
}

// UserIDInt returns the user ID as an int.
func (c *RefreshClaims) UserIDInt() int {
	return c.UserID
}
