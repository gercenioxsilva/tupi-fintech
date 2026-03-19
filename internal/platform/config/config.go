package config

import (
	"os"
)

type Config struct {
	HTTPAddress  string
	LogLevel     string
	DatabasePath string
	Environment  string
}

func Load() Config {
	return Config{
		HTTPAddress:  getEnv("HTTP_ADDRESS", ":8080"),
		LogLevel:     getEnv("LOG_LEVEL", "INFO"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/transactions.jsonl"),
		Environment:  getEnv("APP_ENV", "local"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
