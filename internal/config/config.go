package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const defaultIngestQueue = 256

type Config struct {
	ListenAddr   string
	WALDir       string
	ServicesFile string
	IngestQueue  int
}

func Load() *Config {

	loadDotEnv()

	return &Config{
		ListenAddr:   envOrDefault("GOZZLE_LISTEN", ":8080"),
		WALDir:       envOrDefault("GOZZLE_WAL_DIR", "./data/wal"),
		ServicesFile: envOrDefault("GOZZLE_SERVICES_FILE", "services.json"),
		IngestQueue:  envIntOrDefault("GOZZLE_INGEST_QUEUE", defaultIngestQueue),
	}
}

func loadDotEnv() {

	if _, err := os.Stat(".env"); err != nil {
		return
	}

	if err := godotenv.Load(); err != nil {
		log.Printf("config: warning: load .env: %v", err)
	}
}

func envOrDefault(key, fallback string) string {

	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
