package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NightFury2283/life_forge/internal/app"
	core_auth_jwt "github.com/NightFury2283/life_forge/internal/core/auth/jwt"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	core_postgres_pool "github.com/NightFury2283/life_forge/internal/core/repository/postgres/pool"
	core_http_server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
	"github.com/NightFury2283/life_forge/internal/demo"
	auth_google "github.com/NightFury2283/life_forge/internal/features/auth/repository/google"
	auth_postgres "github.com/NightFury2283/life_forge/internal/features/auth/repository/postgres"
	auth_service "github.com/NightFury2283/life_forge/internal/features/auth/service"
	calendar_google "github.com/NightFury2283/life_forge/internal/features/calendar/repository/google"
	calendar_service "github.com/NightFury2283/life_forge/internal/features/calendar/service"
	gigachat "github.com/NightFury2283/life_forge/internal/features/chat/repository/gigachat"
	chat_service "github.com/NightFury2283/life_forge/internal/features/chat/service"
	stats_postgres "github.com/NightFury2283/life_forge/internal/features/gamification/repository/postgres"
	stats_service "github.com/NightFury2283/life_forge/internal/features/gamification/service"
	users_postgres "github.com/NightFury2283/life_forge/internal/features/users/repository/postgres"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}
	demoMode := flag.Bool("demo", false, "local demo with volatile storage; no Google or PostgreSQL")
	addr := flag.String("addr", envOr("HTTP_ADDR", "127.0.0.1:8080"), "HTTP listen address")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log, err := core_logger.NewLogger(core_logger.LoggerConfig{Level: envOr("LOGGER_LEVEL", "info"), Folder: envOr("LOGGER_FOLDER", "out/logs")})
	if err != nil {
		return err
	}
	defer log.Close()
	secret := os.Getenv("JWT_SECRET_KEY")
	if *demoMode {
		random := make([]byte, 32)
		if _, err := rand.Read(random); err != nil {
			return err
		}
		secret = hex.EncodeToString(random)
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET_KEY must contain at least 32 bytes; configure .env or run with -demo")
	}
	tokens := core_auth_jwt.NewTokenService(core_auth_jwt.JWTConfig{SecretKey: secret})
	var users app.UserRepository
	var stats stats_service.GamificationRepository
	var provider auth_service.GoogleAuthProvider
	options := app.Options{Demo: *demoMode, SecureCookies: !*demoMode && os.Getenv("COOKIE_SECURE") == "true"}
	if *demoMode {
		host, _, err := net.SplitHostPort(*addr)
		if err != nil || (host != "localhost" && host != "127.0.0.1" && host != "::1") {
			return fmt.Errorf("demo must listen on loopback, e.g. -addr 127.0.0.1:8080")
		}
		repository := demo.NewRepository()
		users, stats = repository, repository
		log.Warn("DEMO MODE: shared local account; data resets on restart")
	} else {
		config, err := core_postgres_pool.NewConfig()
		if err != nil {
			return fmt.Errorf("configure PostgreSQL in .env: %w", err)
		}
		connectCtx, stop := context.WithTimeout(ctx, 10*time.Second)
		pool, err := core_postgres_pool.NewConnectionPool(connectCtx, config)
		stop()
		if err != nil {
			return err
		}
		defer pool.Close()
		users = users_postgres.NewUsersRepository(pool)
		statsRepository := stats_postgres.NewGamificationRepository(pool)
		stats = statsRepository
		googleProvider, providerErr := auth_google.NewGoogleAuthProvider(envOr("GOOGLE_CREDENTIALS_PATH", "credentials.json"), os.Getenv("GOOGLE_REDIRECT_URL"))
		provider = googleProvider
		err = providerErr
		if err != nil {
			return err
		}
		store, err := auth_postgres.NewTokenStore(pool, secret)
		if err != nil {
			return err
		}
		options.GoogleTokens = store
		calendar := calendar_service.New(calendar_google.New(googleProvider, store))
		options.Calendar = calendar
		options.Progress = stats_service.NewProgress(statsRepository, calendar)
		if os.Getenv("GIGACHAT_AUTH_KEY") != "" {
			model, err := gigachat.New(gigachat.Config{AuthKey: os.Getenv("GIGACHAT_AUTH_KEY"), Model: envOr("GIGACHAT_MODEL", "GigaChat-Pro"), APIURL: os.Getenv("GIGACHAT_API_URL")})
			if err != nil {
				return err
			}
			options.Chat = chat_service.New(model, calendar)
		}
	}
	handler := app.NewHandler(users, stats, provider, tokens, log, options)
	server := core_http_server.NewHTTPServer(core_http_server.Config{Addr: *addr, ShutdownTimeout: 5 * time.Second}, log)
	server.RegisterMux("/", handler)
	log.Info("LifeForge ready", zap.String("address", "http://"+*addr))
	return server.Run(ctx)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
