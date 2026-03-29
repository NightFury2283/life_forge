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
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Router struct {
	chatHandler     *handlers.ChatHandler
	authHandler     *handlers.AuthHandler
	calendarHandler *handlers.CalendarHandler
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, HX-Request")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := config.New()
	if cfg.GigaChatKey == "" {
		log.Fatal("Couldnt find Gigachat key")
	}

	//signals
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	ai_client := ai.NewGigaChatClient(cfg.GigaChatKey)

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN) //пул соединений с БД
	if err != nil {
		log.Fatal("unable to connect to bd", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("unable to ping db", err)
	}

	contextStorage := storage.NewContextStorage(pool)

	calendarStorage, err := storage.NewGoogleCalendarStorage(pool)
	if err != nil {
		log.Fatal("Error to connect to Google Calendar", err)
	}
	log.Printf("Calendar status: authorized=%v", calendarStorage.IsAuthorized())

	//debug
	if calendarStorage.IsAuthorized() {
		events, err := calendarStorage.ListEvents(context.Background(), time.Now().UTC(), time.Now().AddDate(0, 0, 5).UTC())
		if err != nil {
			log.Printf("Error listing events: %v", err)
		}
		log.Printf("📅 Calendar events found: %d", len(events))
	} else {
		log.Println("Go by url: 🔗 http://localhost:8080/auth/google")
	}

	chatHandler := handlers.NewChatHandler(contextStorage, ai_client, calendarStorage)
	authHandler := handlers.NewAuthHandler(calendarStorage)
	calendarHandler := handlers.NewCalendarHandler(calendarStorage)

	mux := http.NewServeMux()

	router := newRouter(chatHandler, authHandler, calendarHandler)

	router.register(mux)

	handler := corsMiddleware(mux)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Println("Server starting on http://localhost:8080")
	go func() {
		if err := http.ListenAndServe(":8080", handler); err != nil && err != http.ErrServerClosed {
			log.Fatal("Fail Listen and Serve with error ", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down server...")

	shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("Server cannot be stoped. error %w", err)
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

func newRouter(
	chatHandler *handlers.ChatHandler,
	authHandler *handlers.AuthHandler,
	calendarHandler *handlers.CalendarHandler,
) *Router {
	return &Router{
		chatHandler:     chatHandler,
		authHandler:     authHandler,
		calendarHandler: calendarHandler,
	}
}

func (r *Router) register(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("/chat", r.chatHandler.HandleChat)
	mux.HandleFunc("/auth/google", r.authHandler.HandleGoogleLogin)
	mux.HandleFunc("/auth/callback", r.authHandler.HandleGoogleCallback)
	mux.HandleFunc("/api/gantt", r.calendarHandler.HandleGanttDiagramm)
	mux.HandleFunc("/api/calendars", r.calendarHandler.HandleGetCalendars)

	// Init storage context if needed
	//initStorageContext(ctx, storage)

	//mux.HandleFunc("/entry", handler.HandleCreateEntry)
	//mux.HandleFunc("/entries", handler.HandleGetEntries)
}
