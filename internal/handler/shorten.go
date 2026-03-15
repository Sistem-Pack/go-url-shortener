package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

func (h *Shortener) createShortURL(ctx context.Context, originalURL string, userID string) (string, error) {
	parsed, err := url.Parse(originalURL)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("некорректный URL")
	}

	const maxRetries = 3

	for i := range maxRetries {
		id, err := shortid.Generate()
		if err != nil {
			return "", err
		}

		if h.db != nil {
			err = h.db.SaveURL(ctx, id, originalURL, userID)
			if err == nil {
				return url.JoinPath(h.cfg.BaseURL, id)
			}

			var conflictErr *repository.ErrConflict
			if errors.As(err, &conflictErr) {
				return "", err
			}

			log.Warn().Err(err).Int("attempt", i+1).Msg("Коллизия ID или ошибка БД, пробуем снова")
			continue
		} else {
			h.store.Set(id, originalURL)
			return url.JoinPath(h.cfg.BaseURL, id)
		}
	}

	return "", fmt.Errorf("превышено количество попыток генерации уникального ID")

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

		userID, ok := req.Context().Value(config.UserIDKey).(string)
		if !ok {
			http.Error(res, "Unauthorized", http.StatusUnauthorized)
			return
		}

		shortURL, err := h.createShortURL(req.Context(), originalURL, userID)
		if err != nil {
			var conflictErr *repository.ErrConflict

			if errors.As(err, &conflictErr) {
				fullURL, err := url.JoinPath(h.cfg.BaseURL, conflictErr.ShortURL)
				if err != nil {
					http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				res.Header().Set("Content-Type", "text/plain")
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(fullURL))
				return
			}
			http.Error(res, "Ошибка сервера", http.StatusInternalServerError)
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

		userID := request.Context().Value(config.UserIDKey).(string)
		shortURL, err := h.createShortURL(request.Context(), req.URL, userID)
		if err != nil {
			var conflictErr *repository.ErrConflict

			if errors.As(err, &conflictErr) {
				fullURL, err := url.JoinPath(h.cfg.BaseURL, conflictErr.ShortURL)
				if err != nil {
					http.Error(responseWriter, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusConflict)
				json.NewEncoder(responseWriter).Encode(shortenResponse{Result: fullURL})
				return
			}
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

		var originalURL string
		if h.db != nil {
			var err error
			originalURL, err = h.db.GetURL(r.Context(), id)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.Error(w, "Ссылка не найдена", http.StatusNotFound)
					return
				}
				log.Error().Err(err).Str("id", id).Msg("Ошибка при получении URL из БД")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		} else {
			var exists bool
			originalURL, exists = h.store.Get(id)
			if !exists {
				http.Error(w, "Некорректный ID", http.StatusBadRequest)
				return
			}
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func (h *Shortener) PingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.db == nil {
			log.Error().Msg("база данных не инициализирована")

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := h.db.Ping(); err != nil {
			log.Error().Err(err).Msg("не удалось выоплнить проверку соединения с базой данных")

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

func (h *Shortener) GetUserURLs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(config.UserIDKey).(string)

		urls, err := h.db.GetURLsByUserID(r.Context(), userID)
		if err != nil {
			log.Error().Err(err).Msg("Ошибка при получении URL из БД")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if len(urls) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(urls)
	}
}

func (h *Shortener) EncryptUserID(userID string) string {
	hMac := hmac.New(sha256.New, []byte(config.SecretKey))
	hMac.Write([]byte(userID))
	signature := hMac.Sum(nil)

	return hex.EncodeToString(append([]byte(userID), signature...))
}

func (h *Shortener) DecryptUserID(signedValue string) (string, error) {
	data, err := hex.DecodeString(signedValue)
	if err != nil || len(data) < 36 {
		return "", errors.New("некорректная кука")
	}

	userID := string(data[:36])
	signature := data[36:]

	hMac := hmac.New(sha256.New, []byte(config.SecretKey))
	hMac.Write([]byte(userID))
	expectedSignature := hMac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return "", errors.New("неправильная подпись")
	}
	return userID, nil
}

func (h *Shortener) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string
		var setCookie bool

		cookie, err := r.Cookie("user_id")
		if err != nil {
			userID = uuid.New().String()
			setCookie = true
		} else {
			userID, err = h.DecryptUserID(cookie.Value)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		if setCookie {
			signedValue := h.EncryptUserID(userID)
			http.SetCookie(w, &http.Cookie{
				Name:     "user_id",
				Value:    signedValue,
				Path:     "/",
				HttpOnly: true,
			})
		}

		ctx := context.WithValue(r.Context(), config.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NewRouter(cfg *config.Config, store storage.URLStorage, db *repository.PostgresStorage) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.GzipLoggerMiddleware)
	handler := NewShortener(cfg, store, db)
	router.Use(handler.AuthMiddleware)
	router.Post("/", handler.PostHandler())
	router.Post("/api/shorten", handler.PostJSONHandler())
	router.Post("/api/shorten/batch", handler.PostBatchHandler())
	router.Get("/ping", handler.PingHandler())
	router.Get("/{id}", handler.GetHandler())
	router.Get("/api/user/urls", handler.GetUserURLs())
	return router
}
