package main

import (
	"log"
	"plants-monitor/internal/config"
	"plants-monitor/internal/storage"
)

func main() {
	cfg := config.Load()

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer db.Close()
}
