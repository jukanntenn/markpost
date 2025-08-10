package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type JWTTokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func generateJWTToken(userID int, expire time.Duration, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateJWTTokenPair(user *User) (*JWTTokenPair, error) {
	access, err := generateJWTToken(
		user.ID,
		config.JWT.AccessTokenExpire,
		config.JWT.SecretKey,
	)
	if err != nil {
		return nil, err
	}

	refresh, err := generateJWTToken(
		user.ID,
		config.JWT.RefreshTokenExpire,
		config.JWT.SecretKey,
	)
	if err != nil {
		return nil, err
	}

	pair := JWTTokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}

	return &pair, nil
}

func validateJWTToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.JWT.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
