// Package app wires the same application components for the server and tests.
package app

import (
	"net/http"

	core_auth_jwt "github.com/NightFury2283/life_forge/internal/core/auth/jwt"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	core_http_middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
	core_http_response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
	core_http_server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
	auth_service "github.com/NightFury2283/life_forge/internal/features/auth/service"
	auth_http "github.com/NightFury2283/life_forge/internal/features/auth/transport/google/http"
	calendar_http "github.com/NightFury2283/life_forge/internal/features/calendar/transport/http"
	chat_http "github.com/NightFury2283/life_forge/internal/features/chat/transport/http"
	stats_service "github.com/NightFury2283/life_forge/internal/features/gamification/service"
	stats_http "github.com/NightFury2283/life_forge/internal/features/gamification/transport/http"
	users_service "github.com/NightFury2283/life_forge/internal/features/users/service"
	users_http "github.com/NightFury2283/life_forge/internal/features/users/transport/http"
	webassets "github.com/NightFury2283/life_forge/static"
)

type UserRepository interface {
	auth_service.UserRepository
	users_service.UsersRepository
}

type Options struct {
	Demo          bool
	SecureCookies bool
	GoogleTokens  auth_service.GoogleTokenStore
	Calendar      calendar_http.Service
	Chat          chat_http.Service
	Progress      stats_http.ProgressService
}

func NewHandler(users UserRepository, stats stats_service.GamificationRepository,
	provider auth_service.GoogleAuthProvider, tokens *core_auth_jwt.TokenService,
	log *core_logger.Logger, options Options) http.Handler {
	mux := http.NewServeMux()
	api := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	profileHandler := users_http.NewUsersHTTPHandler(users_service.NewUsersService(users))
	statsHandler := stats_http.NewGamificationHTTPHandler(stats_service.NewGamificationService(stats))
	api.RegisterGroup(core_http_server.RouteGroup{
		Middleware: []core_http_middleware.Middleware{core_http_middleware.Auth(tokens)},
		Routes:     append(profileHandler.Routes(), statsHandler.Routes()...),
	})
	if options.Calendar != nil {
		api.RegisterGroup(core_http_server.RouteGroup{Middleware: []core_http_middleware.Middleware{core_http_middleware.Auth(tokens)}, Routes: calendar_http.New(options.Calendar).Routes()})
	}
	if options.Chat != nil {
		api.RegisterGroup(core_http_server.RouteGroup{Middleware: []core_http_middleware.Middleware{core_http_middleware.Auth(tokens)}, Routes: chat_http.New(options.Chat).Routes()})
	}
	if options.Progress != nil {
		api.RegisterGroup(core_http_server.RouteGroup{Middleware: []core_http_middleware.Middleware{core_http_middleware.Auth(tokens)}, Routes: stats_http.NewProgress(options.Progress).Routes()})
	}
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api))
	// Auth endpoints are public; JWT is required only for application data.
	var authService auth_http.AuthService
	if provider != nil {
		authService = auth_service.NewAuthService(provider, users, tokens, options.GoogleTokens)
	}
	authHandler := auth_http.NewAuthHTTPHandler(authService, options.SecureCookies)
	mux.HandleFunc("POST /auth/logout", authHandler.HandleLogout)
	if provider != nil {
		mux.HandleFunc("GET /auth/google", authHandler.HandleGoogleLogin)
		mux.HandleFunc("GET /auth/callback", authHandler.HandleGoogleCallback)
		// Preserve the callback URI registered by the earlier version.
		mux.HandleFunc("GET /api/v1/auth/google", authHandler.HandleGoogleLogin)
		mux.HandleFunc("GET /api/v1/auth/callback", authHandler.HandleGoogleCallback)
	}
	if options.Demo {
		mux.HandleFunc("POST /auth/demo", func(w http.ResponseWriter, r *http.Request) {
			user, err := users.GetOrCreateUserByGoogleID(r.Context(), "local-demo", "demo@localhost", "Исследователь")
			if err != nil {
				http.Error(w, "demo login failed", http.StatusInternalServerError)
				return
			}
			token, err := tokens.GenerateToken(user.ID)
			if err != nil {
				http.Error(w, "token generation failed", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "jwt_token", Value: token, Path: "/", MaxAge: 86400,
				HttpOnly: true, Secure: options.SecureCookies, SameSite: http.SameSiteLaxMode})
			w.WriteHeader(http.StatusNoContent)
		})
	}
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		core_http_response.NewHTTPResponseHanlder(log, w).JSONResponse(map[string]bool{
			"demo": options.Demo, "google_login": provider != nil, "calendar": options.Calendar != nil, "chat": options.Chat != nil, "progress": options.Progress != nil,
		}, http.StatusOK)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(webassets.Page)
	})
	mux.HandleFunc("GET /game", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(webassets.Page)
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(webassets.Files))))
	return core_http_middleware.ChainMiddleware(mux,
		core_http_middleware.RequestID(), core_http_middleware.Logger(log),
		core_http_middleware.Panic(), core_http_middleware.Trace(), browserProtection)
}

func browserProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		// CrossOriginProtection validates browser Origin / Sec-Fetch-Site headers.
		http.NewCrossOriginProtection().Handler(next).ServeHTTP(w, r)
	})
}
