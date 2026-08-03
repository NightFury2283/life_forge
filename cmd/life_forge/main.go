package main

import (
	"context"
	"encoding/json"
	"fmt"
	"life_forge/internal/ai"
	"life_forge/internal/config"
	"life_forge/internal/handlers"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Router struct {
	chatHandler         *handlers.ChatHandler
	authHandler         *handlers.AuthHandler
	calendarHandler     *handlers.CalendarHandler
	gamificationHandler *handlers.GamificationHandler
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

	userStorage := storage.NewUserStorage(pool)

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
	authHandler := handlers.NewAuthHandler(calendarStorage, userStorage)
	calendarHandler := handlers.NewCalendarHandler(calendarStorage)
	// Создаём gamification handler
	gamificationHandler := handlers.NewGamificationHandler(userStorage, calendarStorage)

	mux := http.NewServeMux()

	router := newRouter(chatHandler, authHandler, calendarHandler, gamificationHandler)

	router.register(mux)

	handler := setupMiddleware(corsMiddleware(mux))

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
	gamificationHandler *handlers.GamificationHandler,
) *Router {
	return &Router{
		chatHandler:         chatHandler,
		authHandler:         authHandler,
		calendarHandler:     calendarHandler,
		gamificationHandler: gamificationHandler,
	}
}

func (router *Router) register(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("/chat", router.chatHandler.HandleChat)
	mux.HandleFunc("/auth/google", router.authHandler.HandleGoogleLogin)
	mux.HandleFunc("/auth/callback", router.authHandler.HandleGoogleCallback)
	mux.HandleFunc("/api/gantt", router.calendarHandler.HandleGanttDiagramm)
	mux.HandleFunc("/api/calendars", router.calendarHandler.HandleGetCalendars)

	// Init storage context if needed
	//initStorageContext(ctx, storage)

	//mux.HandleFunc("/entry", handler.HandleCreateEntry)
	//mux.HandleFunc("/entries", handler.HandleGetEntries)

	//gamification
	mux.HandleFunc("/api/profile", router.gamificationHandler.HandleGetProfile)
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			router.gamificationHandler.HandleCreateStat(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/stats/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			router.gamificationHandler.HandleUpdateStat(w, r)
		case http.MethodDelete:
			router.gamificationHandler.HandleDeleteStat(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			router.gamificationHandler.HandleGetTasks(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/tasks/complete", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			router.gamificationHandler.HandleCompleteTask(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/game.html")
	})

	mux.HandleFunc("/setup", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/setup.html")
	})

	mux.HandleFunc("/api/check-google", func(w http.ResponseWriter, r *http.Request) {
		// Проверяем наличие credentials.json
		_, err := os.Stat("credentials.json")
		response := map[string]interface{}{
			"ok":         err == nil,
			"authorized": router.calendarHandler.CalendarStorage.IsAuthorized(),
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/check-gigachat", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Key string }
		json.NewDecoder(r.Body).Decode(&req)

		// Проверяем ключ GigaChat (делаем тестовый запрос)
		client := ai.NewGigaChatClient(req.Key)
		_, err := client.Generate(r.Context(), "Привет")

		response := map[string]interface{}{"ok": err == nil}
		if err != nil {
			response["error"] = err.Error()
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/check-db", func(w http.ResponseWriter, r *http.Request) {
		var config struct {
			Host, Port, User, Password, Dbname string
		}
		json.NewDecoder(r.Body).Decode(&config)

		dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			config.User, config.Password, config.Host, config.Port, config.Dbname)

		pool, err := pgxpool.New(r.Context(), dsn)
		response := map[string]interface{}{"ok": err == nil}
		if err != nil {
			response["error"] = err.Error()
		} else {
			pool.Close()
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/save-config", func(w http.ResponseWriter, r *http.Request) {
		var cfg struct {
			DB struct {
				Host, Port, User, Password, Dbname string
			}
			GigachatKey string
		}
		json.NewDecoder(r.Body).Decode(&cfg)

		// Если пришёл пустой ключ, пробуем сохранить старый
		gigachatKey := cfg.GigachatKey
		if gigachatKey == "" {
			// Читаем существующий .env
			if data, err := os.ReadFile(".env"); err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "GIGACHAT_AUTH_KEY=") {
						gigachatKey = strings.TrimPrefix(line, "GIGACHAT_AUTH_KEY=")
						break
					}
				}
			}
		}

		envContent := fmt.Sprintf(`POSTGRES_DSN=postgres://%s:%s@%s:%s/%s?sslmode=disable
GIGACHAT_AUTH_KEY=%s`,
			cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Dbname,
			gigachatKey)

		err := os.WriteFile(".env", []byte(envContent), 0644)
		if err != nil {
			http.Error(w, "Failed to save", 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
}

// Проверка что настройка завершена (редирект на setup если нет)
func setupMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/setup" || r.URL.Path == "/api/check-setup" {
			next.ServeHTTP(w, r)
			return
		}

		// Проверяем есть ли .env
		if _, err := os.Stat(".env"); os.IsNotExist(err) {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
