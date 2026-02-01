package main

import (
	"log"
	"net/http"

	"github.com/Sistem-Pack/go-url-shortener/internal/app"
	"github.com/Sistem-Pack/go-url-shortener/internal/storage"
	"github.com/Sistem-Pack/go-url-shortener/pkg/config"
)

func main() {
	cfg := config.Init()
	store := storage.NewMemory()
	router := app.New(cfg, store)

	log.Fatal(http.ListenAndServe(cfg.Address, router))
}
