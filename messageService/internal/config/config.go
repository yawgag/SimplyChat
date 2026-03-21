package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	RedisAddr                string
	ServiceAddr              string
	DbURL                    string
	FileServiceURL           string
	FileServiceTimeout       time.Duration
	FileServiceUploadTimeout time.Duration
	FileServiceDeleteTimeout time.Duration
}

func LoadConfig() (*Config, error) {
	config := &Config{
		RedisAddr:                os.Getenv("REDIS_ADDR"),
		ServiceAddr:              os.Getenv("SERVER_ADDRESS"),
		DbURL:                    os.Getenv("DB_URL"),
		FileServiceURL:           valueOrDefault(os.Getenv("FILESERVICE_URL"), "http://file-service:8082"),
		FileServiceTimeout:       parseDurationOrDefault(os.Getenv("FILESERVICE_TIMEOUT"), 10*time.Second),
		FileServiceUploadTimeout: parseDurationOrDefault(os.Getenv("FILESERVICE_UPLOAD_TIMEOUT"), 2*time.Minute),
		FileServiceDeleteTimeout: parseDurationOrDefault(os.Getenv("FILESERVICE_DELETE_TIMEOUT"), 10*time.Second),
	}
	if config.RedisAddr == "" || config.ServiceAddr == "" || config.DbURL == "" {
		return nil, fmt.Errorf("not enough data in config")
	}
	return config, nil
}

func parseDurationOrDefault(raw string, defaultValue time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return defaultValue
	}
	return value
}

func valueOrDefault(value string, defaultValue string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}
	return trimmed
}
