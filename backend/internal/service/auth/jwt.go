package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTTokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AccessClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	accessSigningKey   []byte
	refreshSigningKey  []byte
	accessTokenExpire  time.Duration
	refreshTokenExpire time.Duration
}

func NewJWTService(accessSigningKey, refreshSigningKey string, accessTokenExpire, refreshTokenExpire time.Duration) *JWTService {
	return &JWTService{
		accessSigningKey:   []byte(accessSigningKey),
		refreshSigningKey:  []byte(refreshSigningKey),
		accessTokenExpire:  accessTokenExpire,
		refreshTokenExpire: refreshTokenExpire,
	}
}

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

func (s *JWTService) ValidateAccess(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
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

func (s *JWTService) ValidateRefresh(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
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

func (c *AccessClaims) UserIDInt() int {
	return c.UserID
}

func (c *RefreshClaims) UserIDInt() int {
	return c.UserID
}
