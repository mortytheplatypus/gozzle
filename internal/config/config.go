package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ListenAddr string
	WALDir     string
}

func Load() *Config {

	loadDotEnv()

	return &Config{
		ListenAddr: envOrDefault("GOZZLE_LISTEN", ":8080"),
		WALDir:     envOrDefault("GOZZLE_WAL_DIR", "./data/wal"),
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
