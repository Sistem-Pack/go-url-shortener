package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sistem-Pack/go-url-shortener/internal/middleware"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/go-chi/chi/v5"
	"github.com/teris-io/shortid"
)

type Shortener struct {
	cfg   *config.Config
	store storage.URLStorage
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

func NewShortener(cfg *config.Config, store storage.URLStorage) *Shortener {
	return &Shortener{cfg: cfg, store: store}
}

func (h *Shortener) createShortURL(originalURL string) (string, error) {
	parsed, err := url.Parse(originalURL)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("Некорректный URL")
	}

	id, err := shortid.Generate()
	if err != nil {
		return "", err
	}

	shortURL, err := url.JoinPath(h.cfg.BaseURL, id)
	if err != nil {
		return "", err
	}

	h.store.Set(id, originalURL)

	return shortURL, nil
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

		shortURL, err := h.createShortURL(originalURL)
		if err != nil {
			http.Error(res, "Некорректный URL", http.StatusBadRequest)
			return
		}

		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(shortURL))
	}
}

func (h *Shortener) PostJSONHandler() http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {

		var req shortenRequest

		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			http.Error(responseWriter, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.URL) == "" {
			http.Error(responseWriter, "Пустой URL", http.StatusBadRequest)
			return
		}

		shortURL, err := h.createShortURL(req.URL)
		if err != nil {
			http.Error(responseWriter, "Некорректный URL", http.StatusBadRequest)
			return
		}

		resp := shortenResponse{
			Result: shortURL,
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusCreated)
		json.NewEncoder(responseWriter).Encode(resp)
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

func NewRouter(cfg *config.Config, store storage.URLStorage) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.GzipLoggerMiddleware)
	handler := NewShortener(cfg, store)
	router.Post("/", handler.PostHandler())
	router.Post("/api/shorten", handler.PostJSONHandler())
	router.Get("/{id}", handler.GetHandler())
	return router
}
