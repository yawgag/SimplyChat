package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceAddr        string
	DatabaseURL        string
	S3Bucket           string
	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3UseSSL           bool
	MaxFileSizeBytes   int64
	AllowedMimeTypes   []string
	RequestTimeout     time.Duration
	S3OperationTimeout time.Duration
	S3MaxRetries       int
	PresignedURLTTL    time.Duration
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServiceAddr:        strings.TrimSpace(os.Getenv("SERVER_ADDRESS")),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DB_URL")),
		S3Bucket:           strings.TrimSpace(os.Getenv("S3_BUCKET")),
		S3Endpoint:         strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		S3AccessKey:        strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		S3SecretKey:        strings.TrimSpace(os.Getenv("S3_SECRET_KEY")),
		S3UseSSL:           parseBool(os.Getenv("S3_USE_SSL")),
		MaxFileSizeBytes:   parseInt64WithDefault("MAX_FILE_SIZE_BYTES", 5*1024*1024),
		AllowedMimeTypes:   parseCSV(os.Getenv("ALLOWED_MIME_TYPES")),
		RequestTimeout:     parseDurationWithDefault("REQUEST_TIMEOUT", 10*time.Second),
		S3OperationTimeout: parseDurationWithDefault("S3_OPERATION_TIMEOUT", 10*time.Second),
		S3MaxRetries:       parseIntWithDefault("S3_MAX_RETRIES", 3),
		PresignedURLTTL:    parseDurationWithDefault("PRESIGNED_URL_TTL", 15*time.Minute),
	}

	if cfg.ServiceAddr == "" || cfg.DatabaseURL == "" || cfg.S3Bucket == "" || cfg.S3Endpoint == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		return nil, fmt.Errorf("not enough data in config")
	}
	if cfg.MaxFileSizeBytes <= 0 {
		return nil, fmt.Errorf("max file size must be positive")
	}
	if len(cfg.AllowedMimeTypes) == 0 {
		return nil, fmt.Errorf("allowed mime types list is empty")
	}
	if cfg.S3MaxRetries <= 0 {
		return nil, fmt.Errorf("s3 max retries must be positive")
	}
	if cfg.RequestTimeout <= 0 || cfg.S3OperationTimeout <= 0 || cfg.PresignedURLTTL <= 0 {
		return nil, fmt.Errorf("timeouts must be positive")
	}

	return cfg, nil
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.ToLower(strings.TrimSpace(item))
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func parseBool(raw string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return value
}

func parseIntWithDefault(key string, defaultValue int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return defaultValue
	}
	return value
}

func parseInt64WithDefault(key string, defaultValue int64) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(key)), 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func parseDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return defaultValue
	}
	return value
}
