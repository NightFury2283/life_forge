package main

import (
	"context"
	"fmt"
	"life_forge/internal/ai"
	"life_forge/internal/handlers"
	"life_forge/internal/storage"
	"log"
	"net/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := "postgres://postgres:12345@localhost:5432/life_forge?sslmode=disable"

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
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

	ai_client := ai.NewGigaChatClient("your_key")
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
