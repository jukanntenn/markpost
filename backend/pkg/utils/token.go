package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns the SHA-256 hex digest of the given token string.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
