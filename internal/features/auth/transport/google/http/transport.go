package auth_transport_http

import (
	"context"
	"net/http"

	core_http_server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
)

// Сервис авторизации умеет выдавать ссылки и логинить через разных провайдеров
type AuthService interface {
	GetGoogleAuthURL() string
	LoginViaGoogle(ctx context.Context, code string) (string, error)
	// в будущем можно подключить другие способы логина
}

type AuthHTTPHandler struct {
	authService AuthService
}

func NewAuthHTTPHandler(authService AuthService) *AuthHTTPHandler {
	return &AuthHTTPHandler{
		authService: authService,
	}
}

func (h *AuthHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodGet,
			Path:    "/auth/google",
			Handler: h.HandleGoogleLogin,
		},
		{
			Method:  http.MethodGet,
			Path:    "/auth/callback",
			Handler: h.HandleGoogleCallback,
		},
	}
}
