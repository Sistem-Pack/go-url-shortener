package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Sistem-Pack/go-url-shortener/pkg/shortid"
)

var urlStorage = make(map[string]string)

func checkRequestMethod(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		postProcessing(res, req)
	case http.MethodGet:
		getProcessing(res, req)
	default:
		http.Error(res, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
}

func postProcessing(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil || len(body) == 0 {
		http.Error(res, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	id, _ := shortid.Generate()
	urlStorage[id] = string(body)

	shortURL := fmt.Sprintf("http://localhost:8080/%s", id)

	res.WriteHeader(http.StatusCreated)
	res.Header().Set("Content-Type", "text/plain")
	res.Header().Set("Content-Length", fmt.Sprintf("%d", len(shortURL)))
	res.Write([]byte(shortURL))
}

func getProcessing(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Path[1:]

	originalURL, located := urlStorage[id]
	if !located || id == "" {
		http.Error(res, "Некорректный ID", http.StatusBadRequest)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, checkRequestMethod)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
