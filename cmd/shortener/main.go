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
	router := handler.NewRouter(cfg, store)

	log.Fatal(http.ListenAndServe(cfg.Address, router))
}
