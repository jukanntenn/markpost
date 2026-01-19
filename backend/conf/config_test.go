package conf

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const TEST_ACCESS_SIGNING_KEY = "7e2adc2aacfad5249229fdffee37c0efd7005513bc975e73a95e9c28f47df1a1"
const TEST_REFRESH_SIGNING_KEY = "9c2adc2aacfad5249229fdffee37c0efd7005513bc975e73a95e9c28f47df1b2"

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("MARKPOST_JWT__ACCESS_SIGNING_KEY", TEST_ACCESS_SIGNING_KEY)
	t.Setenv("MARKPOST_JWT__REFRESH_SIGNING_KEY", TEST_REFRESH_SIGNING_KEY)

	ResetForTest()

	if err := LoadConfig(""); err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	cfg := Conf()

	if cfg.Debug != false {
		t.Fatalf("unexpected debug: %v", cfg.Debug)
	}
	if cfg.PostKeyLength != 16 {
		t.Fatalf("unexpected post_key_length: %d", cfg.PostKeyLength)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("unexpected server.host: %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 7330 {
		t.Fatalf("unexpected server.port: %d", cfg.Server.Port)
	}
	expectedProxies := []string{"127.0.0.1", "::1", "localhost"}
	if !reflect.DeepEqual(cfg.Server.TrustedProxies, expectedProxies) {
		t.Fatalf("unexpected server.trusted_proxies: %v", cfg.Server.TrustedProxies)
	}
	if cfg.DB.Driver != "sqlite" {
		t.Fatalf("unexpected db.driver: %s", cfg.DB.Driver)
	}
	if cfg.DB.DSN != "file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL" {
		t.Fatalf("unexpected db.dsn: %s", cfg.DB.DSN)
	}
	if cfg.Admin.InitialUsername != "markpost" || cfg.Admin.InitialPassword != "markpost" {
		t.Fatalf("unexpected admin defaults: %+v", cfg.Admin)
	}
	if cfg.Post.TitleMaxLength != 150 || cfg.Post.BodyMaxBytes != 32768 || cfg.Post.RetentionDays != 7 {
		t.Fatalf("unexpected post defaults: %+v", cfg.Post)
	}
	if !reflect.DeepEqual(cfg.CORS.AllowOrigins, []string{"*"}) {
		t.Fatalf("unexpected cors.allow_origins: %+v", cfg.CORS.AllowOrigins)
	}
	if !reflect.DeepEqual(cfg.CORS.AllowHeaders, []string{"Content-Type", "Authorization", "X-OAuth-State"}) {
		t.Fatalf("unexpected cors.allow_headers: %+v", cfg.CORS.AllowHeaders)
	}
	expectedExposeHeaders := []string{
		"X-Rate-Limit-Limit",
		"X-Rate-Limit-Duration",
		"X-Rate-Limit-Request-Forwarded-For",
		"X-Rate-Limit-Request-Remote-Addr",
		"RateLimit-Limit",
		"RateLimit-Reset",
		"RateLimit-Remaining",
	}
	if !reflect.DeepEqual(cfg.CORS.ExposeHeaders, expectedExposeHeaders) {
		t.Fatalf("unexpected cors.expose_headers: %+v", cfg.CORS.ExposeHeaders)
	}
	if cfg.OAuth.GitHub.ClientID != "" || cfg.OAuth.GitHub.ClientSecret != "" || cfg.OAuth.GitHub.RedirectURL != "" {
		t.Fatalf("unexpected oauth.github defaults: %+v", cfg.OAuth.GitHub)
	}
	if cfg.JWT.AccessSigningKey != TEST_ACCESS_SIGNING_KEY || cfg.JWT.RefreshSigningKey != TEST_REFRESH_SIGNING_KEY {
		t.Fatalf("unexpected jwt signing keys: %s %s", cfg.JWT.AccessSigningKey, cfg.JWT.RefreshSigningKey)
	}
	if cfg.JWT.AccessTokenExpire != 24*time.Hour || cfg.JWT.RefreshTokenExpire != 720*time.Hour {
		t.Fatalf("unexpected jwt defaults: %v", cfg.JWT)
	}
	if cfg.Ratelimit.PerSecond != math.MaxInt || cfg.Ratelimit.Burst != math.MaxInt {
		t.Fatalf("unexpected ratelimit defaults: %+v", cfg.Ratelimit)
	}
}

func TestLoadConfig_PostKeyLengthEnvOverride(t *testing.T) {
	t.Setenv("MARKPOST_JWT__ACCESS_SIGNING_KEY", TEST_ACCESS_SIGNING_KEY)
	t.Setenv("MARKPOST_JWT__REFRESH_SIGNING_KEY", TEST_REFRESH_SIGNING_KEY)
	t.Setenv("MARKPOST_POST_KEY_LENGTH", "18")

	ResetForTest()

	if err := LoadConfig(""); err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	cfg := Conf()
	if cfg.PostKeyLength != 18 {
		t.Fatalf("unexpected post_key_length: %d", cfg.PostKeyLength)
	}
}

func TestLoadConfig_ConfigFileNotExist(t *testing.T) {
	p := filepath.Join(t.TempDir(), "missing.toml")
	ResetForTest()

	err := LoadConfig(p)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected config file not exist error, got: %v", err)
	}
}

func TestLoadConfig_ConfigFileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	content := `
debug = true
[server]
host = "0.0.0.0"
port = 8080
[jwt]
access_signing_key = "` + TEST_ACCESS_SIGNING_KEY + `"
refresh_signing_key = "` + TEST_REFRESH_SIGNING_KEY + `"
access_token_expire = "2h"
refresh_token_expire = "48h"
[post]
title_max_length = 200
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write config file error: %v", err)
	}

	ResetForTest()

	if err := LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	cfg := Conf()

	if cfg.Debug != true {
		t.Fatalf("unexpected debug: %v", cfg.Debug)
	}
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 8080 {
		t.Fatalf("unexpected server: %+v", cfg.Server)
	}
	if cfg.Post.TitleMaxLength != 200 {
		t.Fatalf("unexpected post.title_max_length: %d", cfg.Post.TitleMaxLength)
	}
	if cfg.JWT.AccessSigningKey != TEST_ACCESS_SIGNING_KEY || cfg.JWT.RefreshSigningKey != TEST_REFRESH_SIGNING_KEY {
		t.Fatalf("unexpected jwt signing keys: %s %s", cfg.JWT.AccessSigningKey, cfg.JWT.RefreshSigningKey)
	}
	if cfg.JWT.AccessTokenExpire != 2*time.Hour || cfg.JWT.RefreshTokenExpire != 48*time.Hour {
		t.Fatalf("unexpected jwt durations: %v %v", cfg.JWT.AccessTokenExpire, cfg.JWT.RefreshTokenExpire)
	}
}

func TestLoadConfig_EnvOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	content := `
debug = false
[server]
host = "0.0.0.0"
port = 7330
[jwt]
access_signing_key = "` + TEST_ACCESS_SIGNING_KEY + `"
refresh_signing_key = "` + TEST_REFRESH_SIGNING_KEY + `"
access_token_expire = "1h"
refresh_token_expire = "48h"
[admin]
initial_username = "admin"
initial_password = "admin"
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write config file error: %v", err)
	}

	t.Setenv("MARKPOST_SERVER__HOST", "10.0.0.1")
	t.Setenv("MARKPOST_ADMIN__INITIAL_USERNAME", "admin2")

	ResetForTest()

	if err := LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	cfg := Conf()

	if cfg.Server.Host != "10.0.0.1" {
		t.Fatalf("env override failed for server.host: %s", cfg.Server.Host)
	}
	if cfg.Admin.InitialUsername != "admin2" {
		t.Fatalf("env override failed for admin.initial_username: %s", cfg.Admin.InitialUsername)
	}
}

func TestLoadConfig_ValidationError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	content := `
debug = true
[jwt]
access_signing_key = "short"
refresh_signing_key = "short"
access_token_expire = "1h"
refresh_token_expire = "48h"
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	ResetForTest()

	err := LoadConfig(p)
	if err == nil || !strings.Contains(err.Error(), "failed to validate config") {
		t.Fatalf("expected validate error, got: %v", err)
	}
}

func TestLoadConfig_InvalidRedirectURL(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	content := `
debug = true
[jwt]
access_signing_key = "` + TEST_ACCESS_SIGNING_KEY + `"
refresh_signing_key = "` + TEST_REFRESH_SIGNING_KEY + `"
access_token_expire = "1h"
refresh_token_expire = "48h"
[oauth.github]
redirect_url = "not a url"
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	ResetForTest()

	err := LoadConfig(p)
	if err == nil || !strings.Contains(err.Error(), "failed to validate config") {
		t.Fatalf("expected validate error, got: %v", err)
	}
}

func TestLoadConfig_InvalidPostKeyLength(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	content := `
post_key_length = 11
[jwt]
access_signing_key = "` + TEST_ACCESS_SIGNING_KEY + `"
refresh_signing_key = "` + TEST_REFRESH_SIGNING_KEY + `"
access_token_expire = "1h"
refresh_token_expire = "48h"
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	ResetForTest()

	err := LoadConfig(p)
	if err == nil || !strings.Contains(err.Error(), "failed to validate config") {
		t.Fatalf("expected validate error, got: %v", err)
	}
}
