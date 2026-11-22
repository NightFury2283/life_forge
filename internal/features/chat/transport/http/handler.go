package chat_transport_http

import (
	"context"
	logger "github.com/NightFury2283/life_forge/internal/core/logger"
	middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
	request "github.com/NightFury2283/life_forge/internal/core/transport/http/request"
	response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
	server "github.com/NightFury2283/life_forge/internal/core/transport/http/server"
	chat "github.com/NightFury2283/life_forge/internal/features/chat/service"
	"net/http"
)

type Service interface {
	Ask(context.Context, int, string, []chat.Message) (chat.Reply, error)
}
type Handler struct{ s Service }

func New(s Service) *Handler { return &Handler{s} }
func (h *Handler) Routes() []server.Route {
	return []server.Route{{Method: "POST", Path: "/ai/chat", Handler: h.ask}}
}
func (h *Handler) ask(w http.ResponseWriter, r *http.Request) {
	resp := response.NewHTTPResponseHanlder(logger.FromContext(r.Context()), w)
	var body struct {
		Text    string         `json:"text" validate:"required,max=4000"`
		History []chat.Message `json:"history" validate:"max=12"`
	}
	if err := request.DecodeAndValidateRequest(r, &body); err != nil {
		resp.ErrorResponse(err, "Проверьте сообщение")
		return
	}
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	reply, err := h.s.Ask(r.Context(), userID, body.Text, body.History)
	if err != nil {
		resp.ErrorResponse(err, "Не удалось получить ответ GigaChat")
		return
	}
	resp.JSONResponse(reply, 200)
}
