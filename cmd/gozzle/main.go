package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	go func() {
		log.Printf("server: listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("server: received %s, shutting down", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server: shutdown: %v", err)
	}

	if err := srv.Close(); err != nil {
		log.Printf("server: close: %v", err)
	}

	log.Printf("server: stopped")
}
