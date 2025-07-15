package config

import "os"

type Config struct {
	GatewayAddr     string
	AuthServiceAddr string
}

func LoadConfig() (*Config, error) {
	config := &Config{
		// GatewayAddr:     os.Getenv("SERVER_ADDRESS"),
		// AuthServiceAddr: os.Getenv("AUTH_SERVICE_ADDRESS"),
		AuthServiceAddr: os.Getenv("AUTHSERVICE_ADDR"),
		GatewayAddr:     ":8080",
	}
	return config, nil
}
