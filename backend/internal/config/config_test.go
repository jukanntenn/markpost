package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	ResetForTest()

	err := Load("")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	cfg := Get()
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected default host '127.0.0.1', got %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 7330 {
		t.Fatalf("expected default port 7330, got %d", cfg.Server.Port)
	}
}

func TestGet(t *testing.T) {
	ResetForTest()

	cfg := Get()
	if cfg.Server.Host == "" {
		t.Fatal("expected non-empty host")
	}
}

func TestDefaults(t *testing.T) {
	ResetForTest()

	cfg := Get()

	if cfg.DB.Driver != "sqlite" {
		t.Fatalf("expected default driver 'sqlite', got %s", cfg.DB.Driver)
	}

	if cfg.Post.TitleMaxLength != 150 {
		t.Fatalf("expected default TitleMaxLength 150, got %d", cfg.Post.TitleMaxLength)
	}

	if cfg.Post.BodyMaxBytes != 32768 {
		t.Fatalf("expected default BodyMaxBytes 32768, got %d", cfg.Post.BodyMaxBytes)
	}
}
