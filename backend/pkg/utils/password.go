package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// Password policy per specs/auth.md §4.2/§4.3 (NIST 800-63B):
// min 8 characters, max 72, no forced complexity. The 72 limit is bcrypt's
// byte-level cap, so both the rune count and the byte length are validated.
const (
	PasswordMinRunes = 8
	PasswordMaxRunes = 72
)

// ErrPasswordPolicyViolation indicates the password failed the length policy.
// Callers map it to their domain error (password_too_short / password_too_long).
var ErrPasswordPolicyViolation = errors.New("password violates length policy")

// TooShort reports whether the password is below the minimum length.
func TooShort(password string) bool {
	return utf8.RuneCountInString(password) < PasswordMinRunes
}

// TooLong reports whether the password exceeds the maximum length. bcrypt only
// processes the first 72 bytes, so a password whose byte length exceeds 72
// (e.g. many multi-byte characters) must be rejected even when the rune count
// looks fine.
func TooLong(password string) bool {
	return utf8.RuneCountInString(password) > PasswordMaxRunes || len(password) > PasswordMaxRunes
}

// ValidatePasswordPolicy returns ErrPasswordPolicyViolation when the password
// violates the shared policy (specs/auth.md §4.3: RuneCount + byte double check).
func ValidatePasswordPolicy(password string) error {
	if TooShort(password) {
		return fmt.Errorf("%w: %d < %d runes", ErrPasswordPolicyViolation, utf8.RuneCountInString(password), PasswordMinRunes)
	}
	if TooLong(password) {
		return fmt.Errorf("%w: exceeds %d", ErrPasswordPolicyViolation, PasswordMaxRunes)
	}
	return nil
}

// GenerateRandomPassword returns a cryptographically random password drawn from
// [A-Za-z0-9] (no symbols, to avoid escaping trouble when handed to users).
// K.6: 12 characters ≈ 71 bit entropy.
func GenerateRandomPassword(length int) (string, error) {
	if length <= 0 {
		length = 12
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate random password: %w", err)
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// CheckPassword checks if a password matches a hashed password.
func CheckPassword(password, hashedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("check password: %w", err)
	}
	return true, nil
}
