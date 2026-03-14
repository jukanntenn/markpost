// Package utils provides utility functions.
package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateState generates a random state string for OAuth.
func GenerateState() (string, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
