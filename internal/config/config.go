package config

import (
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	ServerPort       string
	DatabaseType     string // "sqlite", "postgres", "mysql"
	DatabasePath     string // SQLite file path
	DatabaseURL      string // Connection URL for postgres/mysql
	SessionDuration  time.Duration
	UploadMaxSize    int64
	StaticFilesPath  string
	TemplatesPath    string
	MigrationsPath   string
	OAuthRedirectBaseURL string
	GoogleClientID       string
	GoogleClientSecret   string
	FacebookClientID     string
	FacebookClientSecret string
	AppleClientID        string
	AppleClientSecret    string
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		ServerPort:      getEnv("PORT", "8080"),
		DatabaseType:    getEnv("DATABASE_TYPE", "sqlite"),
		DatabasePath:    getEnv("DB_PATH", "./spellingclash.db"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		SessionDuration: 24 * time.Hour,
		UploadMaxSize:   5 * 1024 * 1024, // 5MB
		StaticFilesPath: getEnv("STATIC_PATH", "./static"),
		TemplatesPath:   getEnv("TEMPLATES_PATH", "./internal/templates"),
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "./migrations"),
		OAuthRedirectBaseURL: getEnv("OAUTH_REDIRECT_BASE_URL", ""),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		FacebookClientID:     getEnv("FACEBOOK_CLIENT_ID", ""),
		FacebookClientSecret: getEnv("FACEBOOK_CLIENT_SECRET", ""),
		AppleClientID:        getEnv("APPLE_CLIENT_ID", ""),
		AppleClientSecret:    getEnv("APPLE_CLIENT_SECRET", ""),
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
