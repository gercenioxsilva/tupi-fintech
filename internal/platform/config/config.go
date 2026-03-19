package config

import (
	"os"
)

type Config struct {
	HTTPAddress      string
	LogLevel         string
	Environment      string
	PostgresURL      string
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
}

func Load() Config {
	return Config{
		HTTPAddress:      getEnv("HTTP_ADDRESS", ":8080"),
		LogLevel:         getEnv("LOG_LEVEL", "INFO"),
		Environment:      getEnv("APP_ENV", "local"),
		PostgresURL:      getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/tupi_fintech?sslmode=disable"),
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:       getEnv("POSTGRES_DB", "tupi_fintech"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
