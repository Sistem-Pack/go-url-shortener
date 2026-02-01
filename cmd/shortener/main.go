package main

import (
	"log"
	"net/http"

	"github.com/Sistem-Pack/go-url-shortener/internal/handler"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
)

func main() {
	cfg := config.Init()
	store := storage.NewMemory()
	h := handler.NewShortener(cfg, store)

	router := handler.NewRouter(h)

	log.Fatal(http.ListenAndServe(cfg.Address, router))
}
