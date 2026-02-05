package main

import (
	"context"
	"life_forge/internal/ai"
	"life_forge/internal/config"
	"life_forge/internal/handlers"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.New()
	if cfg.GigaChatKey == "" {
		log.Fatal("Couldnt find Gigachad key")
	}

	ctx := context.Background()
	ai_client := ai.NewGigaChatClient(cfg.GigaChatKey)

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal("unable to connect to bd", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("unable to ping db", err)
	}

	log.Println("connected to db successfully")

	storage := storage.NewContextStorage(pool)

	chatHandler := handlers.NewChatHandler(storage, ai_client)

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", chatHandler.HandleChat)

	// Init storage context if needed
	//initStorageContext(ctx, storage)

	//mux.HandleFunc("/entry", handler.HandleCreateEntry)
	//mux.HandleFunc("/entries", handler.HandleGetEntries)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Fail Listen and Serve with error ", err)
	}
}

func initStorageContext(ctx context.Context, storageInstance *storage.ContextStorage) {
	storageInstance.SaveContext(ctx, &models.Context{
		ID:       1,
		Goals:    []string{},
		Recent5:  []string{},
		Progress: map[string]string{},
	})
}
