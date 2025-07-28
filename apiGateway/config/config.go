package config

import "os"

type Config struct {
	GatewayAddr        string
	AuthServiceAddr    string
	MessageServiceAddr string
}

func LoadConfig() (*Config, error) {
	config := &Config{
		AuthServiceAddr:    os.Getenv("AUTHSERVICE_ADDR"),
		MessageServiceAddr: os.Getenv("MESSAGESERVICE_ADDR"),
		GatewayAddr:        os.Getenv("SERVICE_ADDR"),
	}
	return config, nil
}
