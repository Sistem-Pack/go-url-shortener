package config

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/storage"
	"go-url-shortener/internal/config"
)

func New(cfg *config.Config, store storage.URLStorage) http.Handler {
	r := chi.NewRouter()

	h := handler.NewShortener(cfg, store)
	r.Post("/", h.Shorten)

	return r
}
