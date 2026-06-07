package utils

import (
	"crypto/rand"
	"math/bits"
)

func randomString(byteLength int, alphabet string) (string, error) {
	n := len(alphabet)
	mask := byte(1<<bits.Len8(byte(n-1)) - 1)
	out := make([]byte, 0, byteLength)
	buf := make([]byte, byteLength)
	for len(out) < byteLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			idx := b & mask
			if int(idx) < n {
				out = append(out, alphabet[idx])
				if len(out) == byteLength {
					break
				}
			}
		}
	}
	return string(out), nil
}

// GeneratePostKey creates a cryptographically random post key prefixed with "mpk-".
func GeneratePostKey(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 20
	}
	s, err := randomString(byteLength, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	if err != nil {
		return "", err
	}
	return "mpk-" + s, nil
}

// GenerateRandomPassword creates a cryptographically random password of the given length.
func GenerateRandomPassword(length int) (string, error) {
	if length <= 0 {
		length = 12
	}
	return randomString(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*")
}
