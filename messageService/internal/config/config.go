package config

import (
	"fmt"
	"os"
)

type Config struct {
	RedisAddr   string
	ServiceAddr string
	DbURL       string
}

func LoadConfig() (*Config, error) {
	config := &Config{
		RedisAddr:   os.Getenv("REDIS_ADDR"),
		ServiceAddr: os.Getenv("SERVER_ADDRESS"),
		DbURL:       os.Getenv("DB_URL"),
	}
	if config.RedisAddr == "" || config.ServiceAddr == "" || config.DbURL == "" {
		return nil, fmt.Errorf("not enough data in config")
	}
	return config, nil
}
