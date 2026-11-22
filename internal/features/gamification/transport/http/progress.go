package gamification_transport_http

import (
	"context"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	logger "github.com/NightFury2283/life_forge/internal/core/logger"
	middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
	request "github.com/NightFury2283/life_forge/internal/core/transport/http/request"
	response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
	server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
	"net/http"
)

type ProgressService interface {
	Dashboard(context.Context, int) (domain.HeroDashboard, error)
	Complete(context.Context, int, string, string) (domain.UserProgress, error)
}
type ProgressHandler struct{ s ProgressService }

func NewProgress(s ProgressService) *ProgressHandler { return &ProgressHandler{s} }
func (h *ProgressHandler) Routes() []server.Route {
	return []server.Route{{Method: "GET", Path: "/gamification/dashboard", Handler: h.dashboard}, {Method: "POST", Path: "/gamification/tasks/complete", Handler: h.complete}}
}
func (h *ProgressHandler) dashboard(w http.ResponseWriter, r *http.Request) {
	resp := response.NewHTTPResponseHanlder(logger.FromContext(r.Context()), w)
	id, _ := r.Context().Value(middleware.UserIDKey).(int)
	data, err := h.s.Dashboard(r.Context(), id)
	if err != nil {
		resp.ErrorResponse(err, "Не удалось загрузить прогресс")
		return
	}
	resp.JSONResponse(data, 200)
}
func (h *ProgressHandler) complete(w http.ResponseWriter, r *http.Request) {
	resp := response.NewHTTPResponseHanlder(logger.FromContext(r.Context()), w)
	var body struct {
		EventID    string `json:"event_id" validate:"required,max=1024"`
		CalendarID string `json:"calendar_id" validate:"max=1024"`
	}
	if err := request.DecodeAndValidateRequest(r, &body); err != nil {
		resp.ErrorResponse(err, "Проверьте задачу")
		return
	}
	id, _ := r.Context().Value(middleware.UserIDKey).(int)
	data, err := h.s.Complete(r.Context(), id, body.CalendarID, body.EventID)
	if err != nil {
		resp.ErrorResponse(err, "Задача уже выполнена или недоступна")
		return
	}
	resp.JSONResponse(map[string]any{"profile": data, "xp_gained": 10, "coins_gained": 2}, 200)
}
