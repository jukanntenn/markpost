package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func generateState() (string, error) {
	b := make([]byte, 20) // 160 bits
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func GenerateShortKey(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 8
	}

	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	encoded := base64.URLEncoding.EncodeToString(randomBytes)
	for len(encoded) > 0 && encoded[len(encoded)-1] == '=' {
		encoded = encoded[:len(encoded)-1]
	}

	return encoded, nil
}

// HashPassword 使用bcrypt对密码进行哈希处理
func HashPassword(password string) (string, error) {
	// 使用默认的cost因子，这是当前推荐的值
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// CheckPassword 验证密码是否匹配哈希值
func CheckPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
