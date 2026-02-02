package main

import (
	"context"
	"fmt"
	"life_forge/internal/ai"
	"life_forge/internal/config"
	"life_forge/internal/handlers"
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

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal("unable to connect to bd", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("unable to ping db", err)
	}

	log.Println("connected to db successfully")

	storage := storage.NewJournalStorage(pool)

	handler := handlers.NewJournalHandler(storage)

	mux := http.NewServeMux()

	ai_client := ai.NewGigaChatClient(cfg.GigaChatKey)
	response, err := ai_client.Generate("Доброе утро, что ты хочешь на завтрак?")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(response)

	mux.HandleFunc("/entry", handler.HandleCreateEntry)
	mux.HandleFunc("/entries", handler.HandleGetEntries)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Fail Listen and Serve with error ", err)
	}
}
