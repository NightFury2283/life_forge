package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	PostgresDSN        string
	GigaChatKey        string
	YandexClientID     string
	YandexClientSecret string
}

func New() *Config {
	_ = godotenv.Load()

	return &Config{
		PostgresDSN:        getEnv("POSTGRES_DSN", "postgres://postgres:12345@localhost:5432/life_forge?sslmode=disable"),
		GigaChatKey:        getEnv("GIGACHAT_AUTH_KEY", ""),
		YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
