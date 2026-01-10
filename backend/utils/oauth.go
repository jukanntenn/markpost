package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateState() (string, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
