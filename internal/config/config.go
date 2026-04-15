package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	BotToken    string
	DatabaseURL string
	GeoUsername string
}

// Load reads the .env file (if present) and returns a validated Config.
func Load() (*Config, error) {
	// godotenv.Load is best-effort: missing .env is fine in production
	_ = godotenv.Load()

	cfg := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		GeoUsername: os.Getenv("GEO_USERNAME"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN environment variable is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	if cfg.GeoUsername == "" {
		return nil, fmt.Errorf("GEO_USERNAME environment variable is required")
	}

	return cfg, nil
}
