package main

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	TitleMaxSize int `mapstructure:"TITLE_MAX_SIZE"`
	BodyMaxSize  int `mapstructure:"BODY_MAX_SIZE"`
	APIRateLimit int `mapstructure:"API_RATE_LIMIT"`

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
}

var config Config

func LoadConfig() error {
	config = Config{
		TitleMaxSize: 1000,
		BodyMaxSize:  10485760,
		APIRateLimit: 60,
	}

	// 设置 JWT 默认值
	config.JWT.AccessTokenExpire = 24 * time.Hour
	config.JWT.RefreshTokenExpire = 30 * 24 * time.Hour // 30 days

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

	return nil
}

func configExists() bool {
	_, err := os.Stat("config.toml")
	return !os.IsNotExist(err)
}
