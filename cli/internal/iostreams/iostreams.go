// Package iostreams centralizes the CLI's input and output streams so every
// command reads and writes through one injectable object (mirrors gh's
// pkg/iostreams): tests swap in buffers and toggle TTY flags without a
// subprocess, and behavior differences between terminals and pipes (indented
// vs compact JSON, prompts vs flag errors) have a single home.
package iostreams

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/term"
)

// IOStreams bundles the standard streams with the facts commands need about
// them: whether stdout/stdin are terminals. All v1 output is plain text; the
// TTY facts decide JSON indentation and whether prompting is possible.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	stdoutIsTTY bool
	stdinIsTTY  bool
}

// System returns the streams for a real invocation, deriving TTY-ness from
// the file descriptors.
func System() *IOStreams {
	return &IOStreams{
		In:          os.Stdin,
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
		stdoutIsTTY: term.IsTerminal(int(os.Stdout.Fd())),
		stdinIsTTY:  term.IsTerminal(int(os.Stdin.Fd())),
	}
}

// Test returns non-TTY streams backed by buffers, plus the stdin buffer so
// tests can feed piped input. Override TTY flags with the setters below to
// exercise terminal-only rendering.
func Test() (io *IOStreams, stdin *bytes.Buffer, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	stdin = &bytes.Buffer{}
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	return &IOStreams{
		In:          stdin,
		Out:         stdout,
		ErrOut:      stderr,
		stdoutIsTTY: false,
		stdinIsTTY:  false,
	}, stdin, stdout, stderr
}

func (s *IOStreams) IsStdoutTTY() bool { return s.stdoutIsTTY }

// IsStdinTTY reports whether stdin is attached to a terminal. Piped stdin
// (false) is how commands detect "body provided via stdin".
func (s *IOStreams) IsStdinTTY() bool { return s.stdinIsTTY }

// CanPrompt reports whether an interactive prompt is possible: both stdout
// and stdin must be terminals. Non-interactive callers (scripts, agents, CI)
// get actionable flag errors instead of a hang.
func (s *IOStreams) CanPrompt() bool { return s.stdoutIsTTY && s.stdinIsTTY }

func (s *IOStreams) SetStdoutTTY(v bool) { s.stdoutIsTTY = v }
func (s *IOStreams) SetStdinTTY(v bool)  { s.stdinIsTTY = v }

// ReadAllStdin drains In. It refuses a terminal stdin: reading a TTY would
// block forever waiting for EOF, so callers must offer a flag/file fallback
// before reaching here.
func (s *IOStreams) ReadAllStdin() (string, error) {
	if s.IsStdinTTY() {
		return "", nil
	}
	return s.ReadAll(s.In)
}

func (s *IOStreams) ReadAll(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}
