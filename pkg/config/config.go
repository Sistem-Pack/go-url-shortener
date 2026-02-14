package config

import (
	"flag"
	"os"
)

type Config struct {
	Address string
	BaseURL string
}

func Init() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP-сервер адрес")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "Базовый урл для сокращенных ссылок")

	flag.Parse()

	if envAddress := os.Getenv("SERVER_ADDRESS"); envAddress != "" {
		cfg.Address = envAddress
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	return cfg
}
