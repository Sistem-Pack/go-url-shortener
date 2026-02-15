package main

import (
	"net/http"

	"github.com/Sistem-Pack/go-url-shortener/internal/handler"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Init()
	var store storage.URLStorage
	if cfg.FileStoragePath != "" {
		fs, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Fatal().Err(err).Msg("не удалось прочитать файл с настройками")
		}
		store = fs
	} else {
		store = storage.NewMemory()
	}
	router := handler.NewRouter(cfg, store)

	if err := http.ListenAndServe(cfg.Address, router); err != nil {
		log.Fatal().Err(err).Msg("сервер не запущен")
	}
}
