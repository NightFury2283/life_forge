package calendar_transport_http

import (
	"context"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	logger "github.com/NightFury2283/life_forge/internal/core/logger"
	middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
	request "github.com/NightFury2283/life_forge/internal/core/transport/http/request"
	response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
	server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
	"net/http"
	"strings"
	"time"
)

type Service interface {
	ListCalendars(context.Context, int) ([]domain.Calendar, error)
	ListEvents(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error)
	CreateEvent(context.Context, int, domain.EventDraft) (domain.CalendarEvent, error)
}
type Handler struct{ s Service }

func New(s Service) *Handler { return &Handler{s} }
func (h *Handler) Routes() []server.Route {
	return []server.Route{
		{Method: "GET", Path: "/calendar/calendars", Handler: h.calendars},
		{Method: "GET", Path: "/calendar/events", Handler: h.events},
		{Method: "POST", Path: "/calendar/events", Handler: h.create},
	}
}
func responseFor(w http.ResponseWriter, r *http.Request) *response.HTTPResponseHandler {
	return response.NewHTTPResponseHanlder(logger.FromContext(r.Context()), w)
}
func user(r *http.Request) int { id, _ := r.Context().Value(middleware.UserIDKey).(int); return id }
func (h *Handler) calendars(w http.ResponseWriter, r *http.Request) {
	resp := responseFor(w, r)
	data, err := h.s.ListCalendars(r.Context(), user(r))
	if err != nil {
		resp.ErrorResponse(err, "Не удалось загрузить календари. Попробуйте переподключить Google.")
		return
	}
	resp.JSONResponse(data, 200)
}
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	resp := responseFor(w, r)
	start, e1 := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
	end, e2 := time.Parse(time.RFC3339, r.URL.Query().Get("end"))
	if e1 != nil || e2 != nil {
		resp.ErrorResponse(core_errors.ErrInvalidArgument, "Укажите start и end в формате RFC3339")
		return
	}
	var ids []string
	if value := r.URL.Query().Get("calendars"); value != "" {
		ids = strings.Split(value, ",")
	}
	data, err := h.s.ListEvents(r.Context(), user(r), start, end, ids)
	if err != nil {
		resp.ErrorResponse(err, "Не удалось загрузить события. Попробуйте переподключить Google.")
		return
	}
	resp.JSONResponse(data, 200)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	resp := responseFor(w, r)
	var draft domain.EventDraft
	if err := request.DecodeAndValidateRequest(r, &draft); err != nil {
		resp.ErrorResponse(err, "Проверьте данные события")
		return
	}
	event, err := h.s.CreateEvent(r.Context(), user(r), draft)
	if err != nil {
		resp.ErrorResponse(err, "Не удалось создать событие")
		return
	}
	resp.JSONResponse(event, 201)
}
