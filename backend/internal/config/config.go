// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Debug         bool            `mapstructure:"debug"`
	PostKeyLength int             `mapstructure:"post_key_length" validate:"gte=12"`
	Server        ServerConfig    `mapstructure:"server"`
	DB            DBConfig        `mapstructure:"db"`
	Admin         AdminConfig     `mapstructure:"admin"`
	Post          PostConfig      `mapstructure:"post"`
	CORS          CORSConfig      `mapstructure:"cors"`
	OAuth         OAuthConfig     `mapstructure:"oauth"`
	JWT           JWTConfig       `mapstructure:"jwt"`
	Ratelimit     RatelimitConfig `mapstructure:"ratelimit"`
	Delivery      DeliveryConfig  `mapstructure:"delivery"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Host           string   `mapstructure:"host" validate:"required"`
	Port           uint16   `mapstructure:"port" validate:"required"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	PublicURL      string   `mapstructure:"public_url" validate:"omitempty,url"`
	FrontendURL    string   `mapstructure:"frontend_url" validate:"omitempty,url"`
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
	AccessSigningKey   string        `mapstructure:"access_signing_key"`
	RefreshSigningKey  string        `mapstructure:"refresh_signing_key"`
	AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
	RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
}

// RatelimitConfig holds rate limiting configuration.
type RatelimitConfig struct {
	PerSecond int `mapstructure:"per_second" validate:"gte=0"`
	Burst     int `mapstructure:"burst" validate:"gte=0"`
}

// DeliveryConfig holds delivery-related configuration.
type DeliveryConfig struct {
	BodyPreviewChars int           `mapstructure:"body_preview_chars" validate:"gte=0"`
	RequestTimeout   time.Duration `mapstructure:"request_timeout" validate:"required"`
	RetryCount       int           `mapstructure:"retry_count" validate:"gte=0"`
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
		v.SetConfigName("markpost")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				loadErr = fmt.Errorf("failed to read config file: %w", err)
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
	v.SetDefault("server.frontend_url", "http://127.0.0.1:3000")
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
	v.SetDefault("ratelimit.per_second", math.MaxInt)
	v.SetDefault("ratelimit.burst", math.MaxInt)
	v.SetDefault("delivery.body_preview_chars", 200)
	v.SetDefault("delivery.request_timeout", "5s")
}

// ResetForTest resets the configuration for testing purposes.
func ResetForTest() {
	configInstance = Config{}
	loadConfigOnce = sync.Once{}
	loadErr = nil
	configPath = ""
}
