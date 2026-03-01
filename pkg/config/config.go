package config

import (
	"flag"
	"os"
)

type Config struct {
	Address            string
	BaseURL            string
	FileStoragePath    string
	DBConnectionString string
}

func Init() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP-сервер адрес")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "Базовый урл для сокращенных ссылок")
	flag.StringVar(&cfg.FileStoragePath, "f", "url_storage.json", "Путь к файлу хранения URL")
	flag.StringVar(&cfg.DBConnectionString, "d", "", "Строка подключения к БД")

	flag.Parse()

	if envAddress := os.Getenv("SERVER_ADDRESS"); envAddress != "" {
		cfg.Address = envAddress
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	if envFile := os.Getenv("FILE_STORAGE_PATH"); envFile != "" {
		cfg.FileStoragePath = envFile
	}

	if endDB := os.Getenv("DATABASE_DSN"); endDB != "" {
		cfg.DBConnectionString = endDB
	}

	return cfg
}
