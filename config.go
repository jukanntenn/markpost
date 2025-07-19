package main

import (
	"errors"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	TitleMaxSize int `mapstructure:"TITLE_MAX_SIZE"`
	BodyMaxSize  int `mapstructure:"BODY_MAX_SIZE"`
	APIRateLimit int `mapstructure:"API_RATE_LIMIT"`
}

var config Config

func LoadConfig() error {
	config = Config{
		TitleMaxSize: 1000,
		BodyMaxSize:  10485760,
		APIRateLimit: 60,
	}

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
