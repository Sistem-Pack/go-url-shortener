package main

import (
	"net/http"

	"github.com/Sistem-Pack/go-url-shortener/internal/handler"
	"github.com/Sistem-Pack/go-url-shortener/internal/repository"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Init()
	var store storage.URLStorage
	var db *repository.PostgresStorage
	if cfg.FileStoragePath != "" {
		fs, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Fatal().Err(err).Msg("не удалось прочитать файл с настройками")
		}
		store = fs
	} else {
		store = storage.NewMemory()
	}
	if cfg.DBConnectionString != "" {
		sqlDB, err := repository.OpenDatabase(cfg.DBConnectionString)
		if err != nil {
			log.Warn().
				Err(err).Msg("произошла ошибка при подключении к БД")
		}
		db = &repository.PostgresStorage{DB: sqlDB}
	} else {
		log.Warn().Msg("настройки для подключения к БД небыли получены")
	}
	router := handler.NewRouter(cfg, store, db)

	if err := http.ListenAndServe(cfg.Address, router); err != nil {
		log.Fatal().Err(err).Msg("сервер не запущен")
	}
}
