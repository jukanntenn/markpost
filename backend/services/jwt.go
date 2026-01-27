package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

func (c *Claims) UserID() (int, error) {
	sub := strings.TrimPrefix(c.Subject, "urn:user:")
	id, err := strconv.Atoi(sub)
	if err != nil {
		return 0, fmt.Errorf("invalid subject")
	}
	return id, nil
}

func (c *Claims) IsAdmin() bool {
	return c.Role == "admin"
}

type JWTTokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTService struct {
	accessSigningKey  string
	refreshSigningKey string
	accessExpire      time.Duration
	refreshExpire     time.Duration
}

func NewJWTService(accessKey, refreshKey string, accessExpire, refreshExpire time.Duration) *JWTService {
	return &JWTService{
		accessSigningKey:  accessKey,
		refreshSigningKey: refreshKey,
		accessExpire:      accessExpire,
		refreshExpire:     refreshExpire,
	}
}

func (s *JWTService) generate(userID int, role string, expire time.Duration, signingKey string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("urn:user:%d", userID),
			Issuer:    "markpost",
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(signingKey))
}

func (s *JWTService) GenerateAccessToken(userID int, role string) (string, error) {
	return s.generate(userID, role, s.accessExpire, s.accessSigningKey)
}

func (s *JWTService) GenerateRefreshToken(userID int, role string) (string, error) {
	return s.generate(userID, role, s.refreshExpire, s.refreshSigningKey)
}

func (s *JWTService) GenerateTokenPair(userID int, role string) (*JWTTokenPair, error) {
	access, err := s.GenerateAccessToken(userID, role)
	if err != nil {
		return nil, err
	}
	refresh, err := s.GenerateRefreshToken(userID, role)
	if err != nil {
		return nil, err
	}
	pair := JWTTokenPair{AccessToken: access, RefreshToken: refresh}
	return &pair, nil
}

func (s *JWTService) parse(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.Issuer != "markpost" {
			return nil, fmt.Errorf("parse token: invalid issuer")
		}
		if claims.Subject == "" || !strings.HasPrefix(claims.Subject, "urn:user:") {
			return nil, fmt.Errorf("parse token: invalid subject")
		}
		return claims, nil
	}
	return nil, fmt.Errorf("parse token: invalid token")
}

func (s *JWTService) ValidateAccess(tokenString string) (*Claims, error) {
	return s.parse(tokenString, s.accessSigningKey)
}

func (s *JWTService) ValidateRefresh(tokenString string) (*Claims, error) {
	return s.parse(tokenString, s.refreshSigningKey)
}
