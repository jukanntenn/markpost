// Package domain defines shared domain-level errors and constants.
package domain

import "errors"

// ErrNotFound indicates that the requested record does not exist.
var ErrNotFound = errors.New("record not found")

// Common domain errors for authentication and user management.
var (
	ErrEmailTaken    = errors.New("email is already taken")
	ErrUsernameTaken = errors.New("username is already taken")
	ErrNoPassword    = errors.New("user has no password set")
	ErrBadPassword   = errors.New("invalid password")
)
