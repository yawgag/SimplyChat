package config

type Config struct {
	GatewayAddr     string
	AuthServiceAddr string
}

func LoadConfig() (*Config, error) {
	config := &Config{
		// GatewayAddr:     os.Getenv("SERVER_ADDRESS"),
		// AuthServiceAddr: os.Getenv("AUTH_SERVICE_ADDRESS"),
		GatewayAddr:     ":8080",
		AuthServiceAddr: "",
	}
	return config, nil
}
