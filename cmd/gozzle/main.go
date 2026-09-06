package main

import (
	"log"

	"github.com/mortytheplatypus/gozzle/internal/config"
	"github.com/mortytheplatypus/gozzle/internal/server"
)

func main() {

	cfg := config.Load()
	log.Printf("config: listen=%s wal-dir=%s services-file=%s ingest-queue=%d", cfg.ListenAddr, cfg.WALDir, cfg.ServicesFile, cfg.IngestQueue)

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("server: %v", err)
	}
	defer srv.Close()

	log.Printf("server: listening on %s", cfg.ListenAddr)

	if err := srv.ListenAndServe(); err != nil {
		log.Printf("server: %v", err)
	}
}
