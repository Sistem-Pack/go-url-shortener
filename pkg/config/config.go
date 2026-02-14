package config

import (
	"flag"
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

	return cfg
}
