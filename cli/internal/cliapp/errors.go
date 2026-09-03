package cliapp

import "fmt"

// Exit codes, mirroring gh's contract: 0 success, 1 any failure, 2 user
// cancellation, 4 authentication (fixable by `markpost auth login`).
const (
	ExitOK     = 0
	ExitError  = 1
	ExitCancel = 2
	ExitAuth   = 4
)

// FlagError marks a usage problem (bad flag, missing argument, unknown
// command). Main prints the command's help — the full help when an agent is
// driving, since agents self-correct from flag lists and examples faster than
// from one-line usage.
type FlagError struct {
	Err error
}

func (e *FlagError) Error() string { return e.Err.Error() }
func (e *FlagError) Unwrap() error { return e.Err }

func FlagErrorf(format string, args ...any) error {
	return &FlagError{Err: fmt.Errorf(format, args...)}
}
