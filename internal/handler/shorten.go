package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sistem-Pack/go-url-shortener/internal/middleware"
	"github.com/Sistem-Pack/go-url-shortener/internal/repository"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/go-chi/chi/v5"
	"github.com/teris-io/shortid"
)

type Shortener struct {
	cfg   *config.Config
	store storage.URLStorage
	db    *repository.PostgresStorage
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

type batchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type batchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func NewShortener(cfg *config.Config, store storage.URLStorage, db *repository.PostgresStorage) *Shortener {
	return &Shortener{
		cfg:   cfg,
		store: store,
		db:    db,
	}
}

func (h *Shortener) createShortURL(ctx context.Context, originalURL string) (string, bool, error) {
	parsed, err := url.Parse(originalURL)
	if err != nil || parsed.Host == "" {
		return "", false, fmt.Errorf("некорректный URL")
	}

	id, err := shortid.Generate()
	if err != nil {
		return "", false, err
	}

	var isConflict bool

	if h.db != nil {
		if errors.Is(err, repository.ErrConflict) {
			oldID, errGet := h.db.GetIDByPath(ctx, originalURL)
			if errGet != nil {
				return "", false, errGet
			}
			id = oldID
			isConflict = true
		} else if err != nil {
			return "", false, err
		}
	} else {
		h.store.Set(id, originalURL)
	}

	shortURL, err := url.JoinPath(h.cfg.BaseURL, id)
	if err != nil {
		return "", false, err
	}

	return shortURL, isConflict, nil
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

		shortURL, isConflict, err := h.createShortURL(req.Context(), originalURL)
		if err != nil {
			http.Error(res, "Некорректный URL", http.StatusBadRequest)
			return
		}

		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))
		if isConflict {
			res.WriteHeader(http.StatusConflict)
		} else {
			res.WriteHeader(http.StatusCreated)
		}
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

		shortURL, isConflict, err := h.createShortURL(request.Context(), req.URL)
		if err != nil {
			http.Error(responseWriter, "Некорректный URL", http.StatusBadRequest)
			return
		}

		resp := shortenResponse{
			Result: shortURL,
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		if isConflict {
			responseWriter.WriteHeader(http.StatusConflict)
		} else {
			responseWriter.WriteHeader(http.StatusCreated)
		}
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

		var originalURL string
		var exists bool

		if h.db != nil {
			var err error
			originalURL, err = h.db.GetURL(r.Context(), id)
			exists = (err == nil)
		} else {
			originalURL, exists = h.store.Get(id)
		}

		if !exists {
			http.Error(w, "Некорректный ID", http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func (h *Shortener) PingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.db == nil {
			http.Error(w, "База недоступна", http.StatusInternalServerError)
			return
		}
		if err := h.db.Ping(); err != nil {
			http.Error(w, "База недоступна", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func (h *Shortener) PostBatchHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		var reqData []batchRequest
		if err := json.NewDecoder(req.Body).Decode(&reqData); err != nil {
			http.Error(res, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		respData := make([]batchResponse, 0, len(reqData))
		dbData := make(map[string]string)

		for _, item := range reqData {
			id, err := shortid.Generate()
			if err != nil {
				http.Error(res, "Ошибка генерации ID", http.StatusInternalServerError)
				return
			}

			shortURL, err := url.JoinPath(h.cfg.BaseURL, id)
			if err != nil {
				return
			}

			respData = append(respData, batchResponse{
				CorrelationID: item.CorrelationID,
				ShortURL:      shortURL,
			})

			dbData[id] = item.OriginalURL
		}

		if h.db != nil {
			if err := h.db.SaveBatch(req.Context(), dbData); err != nil {
				http.Error(res, "Ошибка сохранения в БД", http.StatusInternalServerError)
				return
			}
		} else {
			for id, originalURL := range dbData {
				h.store.Set(id, originalURL)
			}
		}

		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)
		json.NewEncoder(res).Encode(respData)
	}
}

func NewRouter(cfg *config.Config, store storage.URLStorage, db *repository.PostgresStorage) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.GzipLoggerMiddleware)
	handler := NewShortener(cfg, store, db)
	router.Post("/", handler.PostHandler())
	router.Post("/api/shorten", handler.PostJSONHandler())
	router.Post("/api/shorten/batch", handler.PostBatchHandler())
	router.Get("/ping", handler.PingHandler())
	router.Get("/{id}", handler.GetHandler())
	return router
}
