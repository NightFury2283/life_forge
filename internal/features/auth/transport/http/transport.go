package auth_transport_http

import "context"

// Сервис авторизации умеет выдавать ссылки и логинить через разных провайдеров
type AuthService interface {
	GetGoogleAithURL() string
	LoginViaGoogle(ctx context.Context, code string) error
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
