package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"life_forge/internal/ai"
	"life_forge/internal/config"
	"life_forge/internal/handlers"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"log"
	"net/http"
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

	contextStorage := storage.NewContextStorage(pool)

	calendarStorage, err := storage.NewGoogleCalendarStorage()
	log.Printf("🔄 Calendar status: authorized=%v, error=%v", calendarStorage.IsAuthorized(), err)

	if calendarStorage.IsAuthorized() {
		events, _ := calendarStorage.ListEvents(1)
		log.Printf("📅 Calendar events found: %d", len(events))
	} else {
		log.Println("🔗 http://localhost:8080/auth/google")
	}

	if err != nil {
		log.Fatal("Error to connect to Google Calendar", err)
	}

	chatHandler := handlers.NewChatHandler(contextStorage, ai_client, calendarStorage)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("/chat", chatHandler.HandleChat)
	authHandler := handlers.NewAuthHandler(calendarStorage)
	mux.HandleFunc("/auth/google", authHandler.HandleGoogleLogin)
	mux.HandleFunc("/auth/callback", authHandler.HandleGoogleCallback)

	// ------------------------------------------

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
