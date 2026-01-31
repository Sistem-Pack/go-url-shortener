package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
	"github.com/Sistem-Pack/go-url-shortener/pkg/shortid"
	"github.com/go-chi/chi/v5"
)

var (
	urlStorage map[string]string
	appConfig  *config.Config
)

func postProcessing(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "Ошибка чтения тела", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(res, "Пустое тело запроса", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		http.Error(res, "Некорректное тело запроса. URL должен начинаться с http:// или https://", http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(originalURL)
	if err != nil || parsed.Host == "" {
		http.Error(res, "Некорректный URL", http.StatusBadRequest)
		return
	}

	id, _ := shortid.Generate()
	urlStorage[id] = originalURL

	//shortURL := fmt.Sprintf("http://localhost:8080/%s", id)
	shortURL := fmt.Sprintf("%s/%s", strings.TrimRight(appConfig.BaseURL, "/"), id)

	res.Header().Set("Content-Type", "text/plain")
	res.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(shortURL))
}

func getProcessing(res http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		http.Error(res, "Некорректный ID", http.StatusBadRequest)
		return
	}

	originalURL, located := urlStorage[id]
	if !located {
		http.Error(res, "Некорректный ID", http.StatusBadRequest)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	urlStorage = make(map[string]string)
	appConfig = config.Init()
	router := chi.NewRouter()
	router.Get("/{id}", getProcessing)
	router.Post("/", postProcessing)

	if err := http.ListenAndServe(appConfig.Address, router); err != nil {
		panic(err)
	}
}
