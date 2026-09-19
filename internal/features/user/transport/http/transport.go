package users_transport_http

import (
	"context"
	"net/http"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_http_server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
)

type UsersHTTPHandler struct {
	usersService UsersService
}

// интерфейс описывает то, что нужно хэндлеру от бизнес-логики
type UsersService interface {
	GetProfile(ctx context.Context, userID int) (*domain.UserProgress, error)
}

func NewUsersHTTPHandler(usersService UsersService) *UsersHTTPHandler {
	return &UsersHTTPHandler{
		usersService: usersService,
	}
}

func (h *UsersHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodGet,
			Path:    "/gamification/profile",
			Handler: h.GetProfile,
		},
	}
}
