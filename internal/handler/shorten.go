package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/Sistem-Pack/go-url-shortener/pkg/shortid"
	"github.com/go-chi/chi/v5"
)

type Shortener struct {
	cfg   *config.Config
	store storage.URLStorage
}

func NewShortener(cfg *config.Config, store storage.URLStorage) *Shortener {
	return &Shortener{cfg: cfg, store: store}
}

func (h *Shortener) PostHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			http.Error(res, "Ошибка чтения тела", http.StatusBadRequest)
			return
		}

		originalURL := strings.TrimSpace(string(body))
		if originalURL == "" {
			http.Error(res, "Пустое тело запроса", http.StatusBadRequest)
			return
		}

		parsed, err := url.Parse(originalURL)
		if err != nil || parsed.Host == "" {
			http.Error(res, "Некорректный URL", http.StatusBadRequest)
			return
		}

		id, _ := shortid.Generate()
		shortURL, err := url.JoinPath(h.cfg.BaseURL, id)
		h.store.Set(id, originalURL)

		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(shortURL))
	}
}

func (h *Shortener) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Некорректный ID", http.StatusBadRequest)
			return
		}

		originalURL, ok := h.store.Get(id)
		if !ok {
			http.Error(w, "Некорректный ID", http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func NewRouter(h *Shortener) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.PostHandler())
	r.Get("/{id}", h.GetHandler())
	return r
}
