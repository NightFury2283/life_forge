package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	PostgresDSN string
	GigaChatKey string
}

func New() *Config {
	_ = godotenv.Load()

	return &Config{
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:12345@localhost:5432/life_forge?sslmode=disable"),
		GigaChatKey: getEnv("GIGACHAT_AUTH_KEY", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
