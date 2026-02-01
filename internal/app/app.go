package app

import (
	"net/http"

	"github.com/Sistem-Pack/go-url-shortener/internal/handler"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/go-chi/chi/v5"
)

func New(cfg *config.Config, store storage.URLStorage) http.Handler {
	r := chi.NewRouter()
	h := handler.NewShortener(cfg, store)
	r.Post("/", h.PostHandler())
	r.Get("/{id}", h.GetHandler())
	return r
}
