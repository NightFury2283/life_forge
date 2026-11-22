package gamification_transport_http

import (
	"context"
	"net/http"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_http_server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
)

type GamificationHTTPHandler struct {
	gamificationService GamificationService
}

type GamificationService interface {
	ListStats(ctx context.Context, userID int) ([]domain.Stat, error)
	DeleteStat(ctx context.Context, userID, statID int) error
	CreateStat(
		ctx context.Context,
		stat domain.Stat,
	) (domain.Stat, error)
}

func NewGamificationHTTPHandler(
	gamificationService GamificationService,
) *GamificationHTTPHandler {
	return &GamificationHTTPHandler{
		gamificationService: gamificationService,
	}
}

func (h *GamificationHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{Method: http.MethodGet, Path: "/gamification/stats", Handler: h.ListStats},
		{Method: http.MethodDelete, Path: "/gamification/stats/{id}", Handler: h.DeleteStat},
		{
			Method:  http.MethodPost,
			Path:    "/gamification/stats",
			Handler: h.CreateStat,
		},
	}
}
