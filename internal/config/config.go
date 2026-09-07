package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	defaultListenAddr    = ":8080"
	defaultWALDir        = "./data/wal"
	defaultServicesFile  = "services.json"
	defaultIngestQueue   = 256
	defaultWALMaxBytes   = 10 * 1024 * 1024 // 10 MiB
	defaultRetentionDays = 3
)

type Config struct {
	ListenAddr    string
	WALDir        string
	ServicesFile  string
	IngestQueue   int
	WALMaxBytes   int
	RetentionDays int
}

func Load() *Config {

	loadDotEnv()

	return &Config{
		ListenAddr:    envOrDefault("GOZZLE_LISTEN", defaultListenAddr),
		WALDir:        envOrDefault("GOZZLE_WAL_DIR", defaultWALDir),
		ServicesFile:  envOrDefault("GOZZLE_SERVICES_FILE", defaultServicesFile),
		IngestQueue:   envIntOrDefault("GOZZLE_INGEST_QUEUE", defaultIngestQueue),
		WALMaxBytes:   envIntOrDefault("GOZZLE_WAL_MAX_BYTES", defaultWALMaxBytes),
		RetentionDays: envIntOrDefault("GOZZLE_RETENTION_DAYS", defaultRetentionDays),
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
