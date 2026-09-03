package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPath(t *testing.T) {
	t.Run("MARKPOST_CONFIG_DIR wins", func(t *testing.T) {
		t.Setenv("MARKPOST_CONFIG_DIR", "/custom/dir")
		got, err := DefaultPath()
		require.NoError(t, err)
		assert.Equal(t, "/custom/dir/config.toml", got)
	})
	t.Run("XDG_CONFIG_HOME next", func(t *testing.T) {
		t.Setenv("MARKPOST_CONFIG_DIR", "")
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		got, err := DefaultPath()
		require.NoError(t, err)
		assert.Equal(t, "/xdg/markpost/config.toml", got)
	})
	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("MARKPOST_CONFIG_DIR", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/tester")
		got, err := DefaultPath()
		require.NoError(t, err)
		assert.Equal(t, "/home/tester/.config/markpost/config.toml", got)
	})
}

func TestLoadMissingFileIsEmptyConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "config.toml"))
	require.NoError(t, err)
	assert.Equal(t, &Config{}, cfg)
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markpost", "config.toml")
	in := &Config{Auth: Auth{
		Server:       "https://mp.example.com",
		UserID:       7,
		Username:     "alice",
		Name:         "Alice",
		Role:         "user",
		Token:        "at-1",
		RefreshToken: "rt-1",
		ExpiresAt:    1750000000,
	}}
	require.NoError(t, Save(path, in))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	dirInfo, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	out, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestNormalizeServer(t *testing.T) {
	valid := map[string]string{
		"https://mp.example.com":     "https://mp.example.com",
		"http://localhost:2053":      "http://localhost:2053",
		"http://localhost:2053/":     "http://localhost:2053",
		"  http://x  ":               "http://x",
		"http://mp.example.com:8443": "http://mp.example.com:8443",
	}
	for raw, want := range valid {
		got, err := NormalizeServer(raw)
		require.NoError(t, err, raw)
		assert.Equal(t, want, got, raw)
	}
	invalid := []string{
		"",
		"mp.example.com",
		"ftp://mp.example.com",
		"https://mp.example.com/posts",
		"https://mp.example.com?q=1",
		"https://mp.example.com#frag",
	}
	for _, raw := range invalid {
		_, err := NormalizeServer(raw)
		assert.Error(t, err, raw)
	}
}

func TestResolveServer(t *testing.T) {
	t.Run("no server anywhere", func(t *testing.T) {
		t.Setenv("MARKPOST_SERVER", "")
		_, err := ResolveServer(&Config{})
		assert.ErrorIs(t, err, ErrNoServer)
	})
	t.Run("env beats file", func(t *testing.T) {
		t.Setenv("MARKPOST_SERVER", "http://env:1")
		cfg := &Config{Auth: Auth{Server: "http://file:2"}}
		got, err := ResolveServer(cfg)
		require.NoError(t, err)
		assert.Equal(t, "http://env:1", got)
	})
	t.Run("file when env unset", func(t *testing.T) {
		t.Setenv("MARKPOST_SERVER", "")
		cfg := &Config{Auth: Auth{Server: "http://file:2"}}
		got, err := ResolveServer(cfg)
		require.NoError(t, err)
		assert.Equal(t, "http://file:2", got)
	})
}

func TestResolveToken(t *testing.T) {
	t.Run("env token wins and marks fromEnv", func(t *testing.T) {
		t.Setenv("MARKPOST_TOKEN", "env-token")
		cfg := &Config{Auth: Auth{Server: "http://s:1", Token: "stored"}}
		token, fromEnv := ResolveToken(cfg, "http://s:1")
		assert.Equal(t, "env-token", token)
		assert.True(t, fromEnv)
	})
	t.Run("stored token only for its own server", func(t *testing.T) {
		t.Setenv("MARKPOST_TOKEN", "")
		cfg := &Config{Auth: Auth{Server: "http://s:1", Token: "stored"}}
		token, fromEnv := ResolveToken(cfg, "http://s:1")
		assert.Equal(t, "stored", token)
		assert.False(t, fromEnv)

		token, _ = ResolveToken(cfg, "http://other:1")
		assert.Empty(t, token, "a session must not leak to another server")
	})
}
