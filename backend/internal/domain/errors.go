package domain

import "errors"

var ErrNotFound = errors.New("record not found")

var (
	ErrEmailTaken    = errors.New("email is already taken")
	ErrUsernameTaken = errors.New("username is already taken")
	ErrNoPassword    = errors.New("user has no password set")
	ErrBadPassword   = errors.New("invalid password")
)
