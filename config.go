package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	TitleMaxSize int `mapstructure:"TITLE_MAX_SIZE"`
	BodyMaxSize  int `mapstructure:"BODY_MAX_SIZE"`
	APIRateLimit int `mapstructure:"API_RATE_LIMIT"`

	Database struct {
		Type string `mapstructure:"type"`
		URL  string `mapstructure:"url"`
	} `mapstructure:"database"`

	GitHub struct {
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
		RedirectURL  string `mapstructure:"redirect_url"`
	} `mapstructure:"github"`

	JWT struct {
		SecretKey          string        `mapstructure:"secret_key"`
		AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
		RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
	} `mapstructure:"jwt"`

	RateLimit struct {
		IP struct {
			PerMinute int `mapstructure:"per_minute"`
			PerDay    int `mapstructure:"per_day"`
		} `mapstructure:"ip"`
		PostKey struct {
			PerMinute int `mapstructure:"per_minute"`
			PerDay    int `mapstructure:"per_day"`
		} `mapstructure:"post_key"`
	} `mapstructure:"rate_limit"`

	DataCleanup struct {
		PostRetentionDays int `mapstructure:"post_retention_days"`
	} `mapstructure:"data_cleanup"`
}

var config Config

func LoadConfig() error {
	config = Config{
		TitleMaxSize: 1000,
		BodyMaxSize:  10485760,
		APIRateLimit: 60,
	}

	config.Database.Type = "sqlite"
	config.Database.URL = "./data/db.sqlite3"

	config.JWT.AccessTokenExpire = 24 * time.Hour
	config.JWT.RefreshTokenExpire = 30 * 24 * time.Hour

	config.RateLimit.IP.PerMinute = 100
	config.RateLimit.IP.PerDay = 1000
	config.RateLimit.PostKey.PerMinute = 10
	config.RateLimit.PostKey.PerDay = 100

	config.DataCleanup.PostRetentionDays = 7

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return err
	}

	if err := viper.Unmarshal(&config); err != nil {
		return errors.New("failed to unmarshal config: " + err.Error())
	}

	config.Database.Type = strings.ToLower(config.Database.Type)

	if err := validateDatabaseConfig(); err != nil {
		return err
	}

	return nil
}

func configExists() bool {
	_, err := os.Stat("config.toml")
	return !os.IsNotExist(err)
}

func validateDatabaseConfig() error {
	dbType := strings.ToLower(config.Database.Type)

	if dbType != "sqlite" && dbType != "postgresql" {
		return fmt.Errorf("unsupported database type '%s'. Supported types: sqlite, postgresql", config.Database.Type)
	}

	if config.Database.URL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}

	if dbType == "sqlite" {
		if err := validateSQLiteURL(config.Database.URL); err != nil {
			return err
		}
	}

	if dbType == "postgresql" {
		if err := validatePostgreSQLURL(config.Database.URL); err != nil {
			return err
		}
	}

	return nil
}

func validateSQLiteURL(url string) error {
	if url == ":memory:" {
		return nil
	}

	if !strings.HasSuffix(url, ".sqlite3") && !strings.HasSuffix(url, ".sqlite") && !strings.HasSuffix(url, ".db") {
		return fmt.Errorf("SQLite database file should have extension .sqlite3, .sqlite, or .db, got: %s", url)
	}

	if !filepath.IsAbs(url) {
		dir := filepath.Dir(url)
		if dir != "." {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("SQLite database directory does not exist: %s", dir)
			}
		}
	}

	return nil
}

func validatePostgreSQLURL(url string) error {
	if !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://") {
		return fmt.Errorf("PostgreSQL URL must start with 'postgres://' or 'postgresql://', got: %s", url)
	}

	if !strings.Contains(url, "@") {
		return fmt.Errorf("PostgreSQL URL must contain credentials (username:password@host), got: %s", url)
	}

	return nil
}