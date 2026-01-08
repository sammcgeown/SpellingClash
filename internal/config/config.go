package config

import (
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	ServerPort       string
	DatabasePath     string
	SessionDuration  time.Duration
	UploadMaxSize    int64
	StaticFilesPath  string
	TemplatesPath    string
	MigrationsPath   string
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		ServerPort:      getEnv("PORT", "8080"),
		DatabasePath:    getEnv("DB_PATH", "./wordclash.db"),
		SessionDuration: 24 * time.Hour,
		UploadMaxSize:   5 * 1024 * 1024, // 5MB
		StaticFilesPath: getEnv("STATIC_PATH", "./static"),
		TemplatesPath:   getEnv("TEMPLATES_PATH", "./internal/templates"),
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "./migrations"),
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
