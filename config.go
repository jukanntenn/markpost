package main

import (
	"github.com/spf13/viper"
	"log"
)

// Config 配置结构
type Config struct {
	TitleMaxSize  int `mapstructure:"TITLE_MAX_SIZE"`
	BodyMaxSize   int `mapstructure:"BODY_MAX_SIZE"`
	APIRateLimit  int `mapstructure:"API_RATE_LIMIT"`
}

var config Config

// LoadConfig 加载配置文件
func LoadConfig() {
	viper.SetConfigName("markpost")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("TITLE_MAX_SIZE", 1000)
	viper.SetDefault("BODY_MAX_SIZE", 10485760) // 10MB
	viper.SetDefault("API_RATE_LIMIT", 60)      // 每分钟60次

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("读取配置文件失败: %v", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	log.Printf("配置已加载: TitleMaxSize=%d, BodyMaxSize=%d, APIRateLimit=%d", 
		config.TitleMaxSize, config.BodyMaxSize, config.APIRateLimit)
} 