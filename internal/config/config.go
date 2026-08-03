package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	PostgresDSN string
	GigaChatKey string
}

// func New() *Config {
// 	_ = godotenv.Load()

// 	return &Config{
// 		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:12345@localhost:5432/life_forge?sslmode=disable"),
// 		GigaChatKey: getEnv("GIGACHAT_AUTH_KEY", ""),
// 	}
// }

func New() *Config {
	_ = godotenv.Load()

	dsn := getEnv("POSTGRES_DSN", "")
	gigachatKey := getEnv("GIGACHAT_AUTH_KEY", "")

	// Если нет настроек, возвращаем пустой конфиг
	return &Config{
		PostgresDSN: dsn,
		GigaChatKey: gigachatKey,
	}
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
