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
	if cfg.DBConnectionString != "" {
		sqlDB, err := repository.OpenDatabase(cfg.DBConnectionString)
		if err != nil {
			log.Warn().Err(err).Msg("не удалось подключиться к базе данных")
		}

		if err := repository.RunMigrations(sqlDB); err != nil {
			log.Warn().Err(err).Msg("не удалось применить миграцию")
		}

		db = &repository.PostgresStorage{DB: sqlDB}
		log.Info().Msg("используется база данных Postgres")
	} else if cfg.FileStoragePath != "" {
		fs, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Warn().Err(err).Msg("не удалось прочитать файл с настройками")
		}
		store = fs
	} else {
		store = storage.NewMemory()
	}

	router := handler.NewRouter(cfg, store, db)

	if err := http.ListenAndServe(cfg.Address, router); err != nil {
		log.Fatal().Err(err).Msg("сервер не запущен")
	}
}
