package iostreams

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestStreamsDefaultNonTTY(t *testing.T) {
	io, stdin, _, _ := Test()
	assert.False(t, io.IsStdoutTTY())
	assert.False(t, io.IsStdinTTY())
	assert.False(t, io.CanPrompt())

	io.SetStdoutTTY(true)
	io.SetStdinTTY(true)
	assert.True(t, io.CanPrompt())
	_ = stdin
}

func TestReadAllStdin(t *testing.T) {
	io, stdin, _, _ := Test()
	stdin.WriteString("hello\nworld")
	got, err := io.ReadAllStdin()
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld", got)

	// A terminal stdin must not be drained: reading it would block forever.
	io.SetStdinTTY(true)
	got, err = io.ReadAllStdin()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestReadAllReader(t *testing.T) {
	io, _, _, _ := Test()
	got, err := io.ReadAll(strings.NewReader("x"))
	require.NoError(t, err)
	assert.Equal(t, "x", got)
}
