package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPostHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()

	postProcessing(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Ожидалось 201, получено %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	shortURL := string(body)

	if !strings.HasPrefix(shortURL, "http://localhost:8080/") {
		t.Fatalf("Неправильный url: %s", shortURL)
	}
}

func TestGetHandler(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/", postProcessing)
	router.Get("/{id}", getProcessing)

	urlStorage["abc123"] = "https://practicum.yandex.ru/"

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

func TestPostProcessingEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	postProcessing(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Ожидалось %d, получено %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestGetProcessingWrongID(t *testing.T) {
	urlStorage = make(map[string]string)

	req := httptest.NewRequest(http.MethodGet, "/wrong", nil)
	w := httptest.NewRecorder()

	getProcessing(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Ожидалось %d, получено %d", http.StatusBadRequest, resp.StatusCode)
	}
}
