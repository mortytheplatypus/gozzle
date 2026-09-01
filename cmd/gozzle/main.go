package main

import (
	"log"

	"github.com/mortytheplatypus/gozzle/internal/config"
	"github.com/mortytheplatypus/gozzle/internal/server"
)

func main() {

	cfg := config.Load()
	log.Printf("config: listen=%s wal-dir=%s", cfg.ListenAddr, cfg.WALDir)

	srv := server.New(cfg)
	log.Printf("server: listening on %s", cfg.ListenAddr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
