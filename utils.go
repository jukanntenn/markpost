package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func GenerateState() (string, error) {
	b := make([]byte, 20) // 160 bits
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func GeneratePostKey(byteLength int) (string, error) {
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

func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

func CheckPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
