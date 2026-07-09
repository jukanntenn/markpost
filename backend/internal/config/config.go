// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Debug         bool             `mapstructure:"debug"`
	PostKeyLength int              `mapstructure:"post_key_length" validate:"gte=12"`
	Server        ServerConfig     `mapstructure:"server"`
	DB            DBConfig         `mapstructure:"db"`
	Admin         AdminConfig      `mapstructure:"admin"`
	Post          PostConfig       `mapstructure:"post"`
	CORS          CORSConfig       `mapstructure:"cors"`
	OAuth         OAuthConfig      `mapstructure:"oauth"`
	JWT           JWTConfig        `mapstructure:"jwt"`
	Ratelimit     RatelimitConfig  `mapstructure:"ratelimit"`
	Delivery      DeliveryConfig   `mapstructure:"delivery"`
	Render        RenderConfig     `mapstructure:"render"`
	Cloudflare    CloudflareConfig `mapstructure:"cloudflare"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Host           string   `mapstructure:"host" validate:"required"`
	Port           uint16   `mapstructure:"port" validate:"required"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	PublicURL      string   `mapstructure:"public_url" validate:"omitempty,url"`
}

// DBConfig holds database-related configuration.
type DBConfig struct {
	Driver string `mapstructure:"driver" validate:"oneof=sqlite mysql postgresql"`
	DSN    string `mapstructure:"dsn" validate:"required"`
}

// AdminConfig holds admin-related configuration.
type AdminConfig struct {
	InitialUsername string `mapstructure:"initial_username" validate:"required"`
	InitialPassword string `mapstructure:"initial_password" validate:"required"`
}

// PostConfig holds post-related configuration.
type PostConfig struct {
	TitleMaxLength int `mapstructure:"title_max_length" validate:"gte=0"`
	BodyMaxBytes   int `mapstructure:"body_max_bytes" validate:"gte=0"`
	RetentionDays  int `mapstructure:"retention_days" validate:"gte=0"`
}

// CORSConfig holds CORS-related configuration.
type CORSConfig struct {
	AllowOrigins  []string `mapstructure:"allow_origins"`
	AllowHeaders  []string `mapstructure:"allow_headers"`
	ExposeHeaders []string `mapstructure:"expose_headers"`
}

// OAuthConfig holds OAuth-related configuration.
type OAuthConfig struct {
	GitHub GitHubOAuthConfig `mapstructure:"github"`
}

// GitHubOAuthConfig holds GitHub OAuth configuration.
type GitHubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url" validate:"omitempty,url"`
}

// JWTConfig holds JWT-related configuration.
type JWTConfig struct {
	AccessSigningKey   string        `mapstructure:"access_signing_key" validate:"required"`
	RefreshSigningKey  string        `mapstructure:"refresh_signing_key" validate:"required"`
	AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
	RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
}

// RatelimitConfig holds rate limiting configuration for the three independent
// limiters. Each limiter is scoped to a route class and keyed on the dimension
// that actually identifies the actor (IP for the public read path, user_id for
// the authenticated write paths). Rates are per second; burst is the token
// bucket capacity. L2 carries an additional daily cap expressed as a per-second
// rate (1000/86400).
type RatelimitConfig struct {
	Read RateLimitConfig `mapstructure:"read"`
	L2   RateLimitConfig `mapstructure:"public_write"`
	L3   RateLimitConfig `mapstructure:"authed_write"`
}

// RateLimitConfig holds a single limiter's rate and burst, plus an optional
// secondary daily-cap rate (used only by L2's public-write limiter).
type RateLimitConfig struct {
	PerSecond   float64 `mapstructure:"per_second" validate:"gte=0"`
	Burst       int     `mapstructure:"burst" validate:"gte=0"`
	DailyPerSec float64 `mapstructure:"daily_per_second" validate:"gte=0"`
	DailyBurst  int     `mapstructure:"daily_burst" validate:"gte=0"`
}

// DeliveryConfig holds delivery-related configuration.
type DeliveryConfig struct {
	BodyPreviewChars int           `mapstructure:"body_preview_chars" validate:"gte=0"`
	RequestTimeout   time.Duration `mapstructure:"request_timeout" validate:"required"`
	RetryCount       int           `mapstructure:"retry_count" validate:"gte=0"`
}

// RenderConfig holds configuration for the in-process render cache
// (singleflight + ristretto). The cache stores rendered HTML/raw bodies keyed
// by QID + buildID; entries are invalidated on delete/prune and rotated on
// release. A small or self-hosted instance can disable or shrink it.
type RenderConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	CacheSizeBytes int  `mapstructure:"cache_size_bytes" validate:"gte=0"`
	NumCounters    int  `mapstructure:"num_counters" validate:"gte=0"`
	BufferItems    int  `mapstructure:"buffer_items" validate:"gte=0"`
}

// CloudflareConfig holds the optional Cloudflare API credentials used for
// best-effort cache-tag purge on post deletion. Absent (the default) means the
// instance is self-hosted without Cloudflare; deletion still removes the origin
// render cache and DB row, and the CDN falls back to natural TTL expiry.
type CloudflareConfig struct {
	APIToken string `mapstructure:"api_token"`
	ZoneID   string `mapstructure:"zone_id"`
}

var (
	configInstance Config
	loadConfigOnce sync.Once
	loadErr        error
	configPath     string
)

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func loadConfig() {
	v := viper.New()

	setDefaults(v)

	if configPath != "" {
		exists, err := fileExists(configPath)
		if err != nil {
			loadErr = fmt.Errorf("failed to check config file: %w", err)
			return
		}
		if !exists {
			loadErr = fmt.Errorf("config file does not exist: %s", configPath)
			return
		}
		v.SetConfigFile(configPath)
		v.SetConfigType("toml")
		if err := v.ReadInConfig(); err != nil {
			loadErr = fmt.Errorf("failed to read config file: %w", err)
			return
		}
	} else {
		v.SetConfigType("toml")
		v.AddConfigPath(".")
		var readErr error
		for _, name := range []string{"config", "markpost"} {
			v.SetConfigName(name)
			readErr = v.ReadInConfig()
			if readErr == nil {
				break
			}
			if _, ok := readErr.(viper.ConfigFileNotFoundError); !ok {
				loadErr = fmt.Errorf("failed to read config file: %w", readErr)
				return
			}
		}
	}

	v.SetEnvPrefix("MARKPOST")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	if err := v.Unmarshal(&configInstance); err != nil {
		loadErr = fmt.Errorf("failed to unmarshal config: %w", err)
		return
	}

	validate := validator.New()
	if err := validate.Struct(&configInstance); err != nil {
		loadErr = fmt.Errorf("failed to validate config: %w", err)
		return
	}
}

// Load loads the configuration from the specified path.
func Load(path string) error {
	configPath = path
	loadConfigOnce.Do(loadConfig)
	return loadErr
}

// Get returns the loaded configuration.
func Get() Config {
	loadConfigOnce.Do(loadConfig)
	return configInstance
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("debug", false)
	v.SetDefault("post_key_length", 16)
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 7330)
	v.SetDefault("server.trusted_proxies", []string{"127.0.0.1", "::1"})
	v.SetDefault("server.public_url", "")
	v.SetDefault("db.driver", "sqlite")
	v.SetDefault("db.dsn", "file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL")
	v.SetDefault("admin.initial_username", "markpost")
	v.SetDefault("admin.initial_password", "markpost")
	v.SetDefault("post.title_max_length", 150)
	v.SetDefault("post.body_max_bytes", 32768)
	v.SetDefault("post.retention_days", 7)
	v.SetDefault("cors.allow_origins", []string{"*"})
	v.SetDefault("cors.allow_headers", []string{"Content-Type", "Authorization", "X-OAuth-State"})
	v.SetDefault("cors.expose_headers", []string{
		"X-Rate-Limit-Limit",
		"X-Rate-Limit-Duration",
		"X-Rate-Limit-Request-Forwarded-For",
		"X-Rate-Limit-Request-Remote-Addr",
		"RateLimit-Limit",
		"RateLimit-Reset",
		"RateLimit-Remaining",
	})
	v.SetDefault("oauth.github.client_id", "")
	v.SetDefault("oauth.github.client_secret", "")
	v.SetDefault("oauth.github.redirect_url", "")
	v.SetDefault("jwt.access_signing_key", "")
	v.SetDefault("jwt.refresh_signing_key", "")
	v.SetDefault("jwt.access_token_expire", "24h")
	v.SetDefault("jwt.refresh_token_expire", "720h")
	// L1 read: per-IP, generous — the CDN absorbs most reads; this only governs
	// the small fraction that revalidates against the origin.
	v.SetDefault("ratelimit.read.per_second", 100)
	v.SetDefault("ratelimit.read.burst", 200)
	// L2 public write: per user_id (resolved by PostKey). 10/min + a 1000/day cap,
	// matching the business hard limits.
	v.SetDefault("ratelimit.public_write.per_second", 0.1666666667)
	v.SetDefault("ratelimit.public_write.burst", 20)
	v.SetDefault("ratelimit.public_write.daily_per_second", 1000.0/86400)
	v.SetDefault("ratelimit.public_write.daily_burst", 1000)
	// L3 authenticated write: per user_id from the JWT. 30/min.
	v.SetDefault("ratelimit.authed_write.per_second", 0.5)
	v.SetDefault("ratelimit.authed_write.burst", 60)
	v.SetDefault("delivery.body_preview_chars", 200)
	v.SetDefault("delivery.request_timeout", "5s")
	v.SetDefault("delivery.retry_count", 0)
	v.SetDefault("render.enabled", true)
	v.SetDefault("render.cache_size_bytes", 134217728) // 128 MiB
	v.SetDefault("render.num_counters", 100000)        // ~10x expected key count
	v.SetDefault("render.buffer_items", 64)
}

// ResetForTest resets the configuration for testing purposes.
func ResetForTest() {
	configInstance = Config{}
	loadConfigOnce = sync.Once{}
	loadErr = nil
	configPath = ""
}
