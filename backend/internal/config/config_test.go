package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testConfigToml = `
server.host = "127.0.0.1"
server.port = 7330
[db]
driver = "sqlite"
dsn = ":memory:"
[admin]
initial_username = "markpost"
initial_password = "markpost"
[jwt]
access_signing_key = "test-access-key-at-least-32-characters"
refresh_signing_key = "test-refresh-key-at-least-32-characters"
[delivery]
request_timeout = "5s"
`

func writeTestConfig(t *testing.T) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-config-*.toml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, _ = tmpFile.WriteString(testConfigToml)
	_ = tmpFile.Close()
	return tmpFile.Name()
}

func TestLoad(t *testing.T) {
	ResetForTest()

	path := writeTestConfig(t)
	defer func() { _ = os.Remove(path) }()

	err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	cfg := Get()
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected host '127.0.0.1', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 7330 {
		t.Fatalf("expected port 7330, got %d", cfg.Server.Port)
	}
}

func TestLoad_WithoutConfigFails(t *testing.T) {
	ResetForTest()

	err := Load("")
	if err == nil {
		t.Fatal("expected error when no config file is provided")
	}
}

func TestGet(t *testing.T) {
	ResetForTest()

	path := writeTestConfig(t)
	defer func() { _ = os.Remove(path) }()

	if err := Load(path); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	cfg := Get()
	if cfg.Server.Host == "" {
		t.Fatal("expected non-empty host")
	}
}

func TestDefaults(t *testing.T) {
	ResetForTest()

	path := writeTestConfig(t)
	defer func() { _ = os.Remove(path) }()

	if err := Load(path); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	cfg := Get()
	if cfg.Post.TitleMaxLength != 150 {
		t.Fatalf("expected default TitleMaxLength 150, got %d", cfg.Post.TitleMaxLength)
	}
	if cfg.Post.BodyMaxBytes != 32768 {
		t.Fatalf("expected default BodyMaxBytes 32768, got %d", cfg.Post.BodyMaxBytes)
	}
}

func TestFileExists(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-config")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		_ = tmpFile.Close()
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		exists, err := fileExists(tmpFile.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		exists, err := fileExists("/nonexistent/path/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected file to not exist")
		}
	})
}

func TestLoad_WithInvalidPath(t *testing.T) {
	ResetForTest()

	err := Load("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("expected error for non-existent config file")
	}
}

func TestLoad_WithValidTomlFile(t *testing.T) {
	ResetForTest()

	tmpFile, err := os.CreateTemp("", "test-config-*.toml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	content := `
debug = true
post_key_length = 20
[server]
host = "0.0.0.0"
port = 8080
[db]
driver = "sqlite"
dsn = ":memory:"
[admin]
initial_username = "admin"
initial_password = "secret"
[jwt]
access_signing_key = "test-access-key-min-32-characters!!"
refresh_signing_key = "test-refresh-key-min-32-characters!!"
[delivery]
request_timeout = "5s"
`
	_, _ = tmpFile.WriteString(content)
	_ = tmpFile.Close()

	err = Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := Get()
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Server.Port)
	}
}

func writeNamedConfig(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(testConfigToml), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestLoad_AutoDiscoveryConfigToml(t *testing.T) {
	ResetForTest()

	t.Chdir(t.TempDir())
	writeNamedConfig(t, ".", "config.toml")

	if err := Load(""); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	cfg := Get()
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected host '127.0.0.1', got %s", cfg.Server.Host)
	}
}

func TestLoad_AutoDiscoveryMarkpostTomlFallback(t *testing.T) {
	ResetForTest()

	t.Chdir(t.TempDir())
	writeNamedConfig(t, ".", "markpost.toml")

	if err := Load(""); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	cfg := Get()
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected host '127.0.0.1', got %s", cfg.Server.Host)
	}
}
