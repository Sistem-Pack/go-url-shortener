package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sistem-Pack/go-url-shortener/internal/handler"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/go-chi/chi/v5"
)

func TestPostHandler(t *testing.T) {
	store := storage.NewMemory()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	h := handler.NewShortener(cfg, store)
	handler := h.PostHandler()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Ожидалось 201, получено %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Не удалось прочитать тело запроса")
	}

	shortURL := string(body)

	if !strings.HasPrefix(shortURL, "http://localhost:8080/") {
		t.Fatalf("Неправильный url: %s", shortURL)
	}
}

func TestGetHandler(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	store := storage.NewMemory()
	store.Set("abc123", "https://practicum.yandex.ru/")
	h := handler.NewShortener(cfg, store)

	router := chi.NewRouter()
	router.Get("/{id}", h.GetHandler())

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Ожидалось 307, получено %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "https://practicum.yandex.ru/" {
		t.Fatalf("Неожиданный Location заголовок: %s", location)
	}
}

func TestPostHandlerEmptyBody(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	store := storage.NewMemory()
	h := handler.NewShortener(cfg, store)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	h.PostHandler()(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Ожидалось %d, получено %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestGetHandlerWrongID(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	store := storage.NewMemory()

	router := handler.NewRouter(cfg, store)

	req := httptest.NewRequest(http.MethodGet, "/wrong", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Ожидалось %d, получено %d", http.StatusBadRequest, resp.StatusCode)
	}
}
