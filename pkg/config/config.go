package config

import (
	"flag"
	"os"
)

type ContextKey string

const UserIDKey ContextKey = "userID"
const SecretKey = "9vU2OWQPdG7wnNYNb5WoTEkT5T3HiwM9hqpLLwBWm78="

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

	envAddress, exists := os.LookupEnv("SERVER_ADDRESS")
	if exists {
		cfg.Address = envAddress
	}

	envBaseURL, exists := os.LookupEnv("BASE_URL")
	if exists {
		cfg.BaseURL = envBaseURL
	}

	envFile, exists := os.LookupEnv("FILE_STORAGE_PATH")
	if exists {
		cfg.FileStoragePath = envFile
	}

	endDB, exists := os.LookupEnv("DATABASE_DSN")
	if exists {
		cfg.DBConnectionString = endDB
	}

	return cfg
}
