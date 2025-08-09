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

	// 数据库配置
	Database struct {
		Type string `mapstructure:"type"`
		URL  string `mapstructure:"url"`
	} `mapstructure:"database"`

	// GitHub OAuth2 配置
	GitHub struct {
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
		RedirectURL  string `mapstructure:"redirect_url"`
	} `mapstructure:"github"`

	// JWT 配置
	JWT struct {
		SecretKey          string        `mapstructure:"secret_key"`
		AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
		RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
	} `mapstructure:"jwt"`

	// 限流配置
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

	// 数据清理配置
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

	// 设置数据库默认值
	config.Database.Type = "sqlite"
	config.Database.URL = "./data/db.sqlite3"

	// 设置 JWT 默认值
	config.JWT.AccessTokenExpire = 24 * time.Hour
	config.JWT.RefreshTokenExpire = 30 * 24 * time.Hour // 30 days

	// 设置限流默认值
	config.RateLimit.IP.PerMinute = 100
	config.RateLimit.IP.PerDay = 1000
	config.RateLimit.PostKey.PerMinute = 10
	config.RateLimit.PostKey.PerDay = 100

	// 设置数据清理默认值
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

	// 标准化数据库类型（转为小写）
	config.Database.Type = strings.ToLower(config.Database.Type)

	// 验证数据库配置
	if err := validateDatabaseConfig(); err != nil {
		return err
	}

	return nil
}

func configExists() bool {
	_, err := os.Stat("config.toml")
	return !os.IsNotExist(err)
}

// validateDatabaseConfig 验证数据库配置
func validateDatabaseConfig() error {
	dbType := strings.ToLower(config.Database.Type)

	// 验证数据库类型
	if dbType != "sqlite" && dbType != "postgresql" {
		return fmt.Errorf("unsupported database type '%s'. Supported types: sqlite, postgresql", config.Database.Type)
	}

	// 验证 URL 不为空
	if config.Database.URL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}

	// 验证 SQLite 配置
	if dbType == "sqlite" {
		if err := validateSQLiteURL(config.Database.URL); err != nil {
			return err
		}
	}

	// 验证 PostgreSQL 配置
	if dbType == "postgresql" {
		if err := validatePostgreSQLURL(config.Database.URL); err != nil {
			return err
		}
	}

	return nil
}

// validateSQLiteURL 验证 SQLite URL 格式
func validateSQLiteURL(url string) error {
	// 允许内存数据库
	if url == ":memory:" {
		return nil
	}

	// 检查文件路径是否合理
	if !strings.HasSuffix(url, ".sqlite3") && !strings.HasSuffix(url, ".sqlite") && !strings.HasSuffix(url, ".db") {
		return fmt.Errorf("SQLite database file should have extension .sqlite3, .sqlite, or .db, got: %s", url)
	}

	// 如果是相对路径，检查目录是否存在
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

// validatePostgreSQLURL 验证 PostgreSQL URL 格式
func validatePostgreSQLURL(url string) error {
	// 基本的 PostgreSQL URL 格式检查
	if !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://") {
		return fmt.Errorf("PostgreSQL URL must start with 'postgres://' or 'postgresql://', got: %s", url)
	}

	// 检查 URL 是否包含基本组件
	if !strings.Contains(url, "@") {
		return fmt.Errorf("PostgreSQL URL must contain credentials (username:password@host), got: %s", url)
	}

	return nil
}
