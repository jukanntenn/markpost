package utils

import "crypto/rand"

func GeneratePostKey(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 20
	}
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	n := len(alphabet)
	out := make([]byte, byteLength)
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := 0; i < byteLength; i++ {
		out[i] = alphabet[int(buf[i])%n]
	}
	return "mpk-" + string(out), nil
}
