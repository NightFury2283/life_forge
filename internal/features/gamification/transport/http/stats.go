package gamification_transport_http

import (
	"net/http"
	"strconv"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	core_http_middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
	core_http_request "github.com/NightFury2283/life_forge/internal/core/transport/http/request"
	core_http_response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
)

type CreateStatRequest struct {
	Title       string  `json:"title" validate:"required,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,min=1,max=1000"`
	Value       int     `json:"value" validate:"omitempty,min=0"`
}

type CreateStatResponse struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Value       int     `json:"value"`
}

func (h *GamificationHTTPHandler) CreateStat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHanlder(log, w)

	userID, ok := ctx.Value(core_http_middleware.UserIDKey).(int)

	if !ok || userID == 0 {
		responseHandler.ErrorResponse(core_errors.ErrUnauthorized, "Unauthorized")
		return
	}

	log.Debug("Method CreateStat start")

	var request CreateStatRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	statDomain := domain.NewStatUnitialized(
		userID,
		request.Title,
		request.Description,
		request.Value,
	)

	respStatDomain, err := h.gamificationService.CreateStat(ctx, statDomain)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create stat")
		return
	}

	statResponse := statDTOFromDomain(respStatDomain)

	responseHandler.JSONResponse(statResponse, http.StatusCreated)
}

func statDTOFromDomain(stat domain.Stat) CreateStatResponse {
	return CreateStatResponse{
		ID:          stat.ID,
		Title:       stat.Title,
		Description: stat.Description,
		Value:       stat.Value,
	}
}

func (h *GamificationHTTPHandler) AdjustStat(w http.ResponseWriter, r *http.Request) {
	response := core_http_response.NewHTTPResponseHanlder(core_logger.FromContext(r.Context()), w)
	userID, _ := r.Context().Value(core_http_middleware.UserIDKey).(int)
	statID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.ErrorResponse(core_errors.ErrInvalidArgument, "Неверный навык")
		return
	}
	var body struct {
		Delta int `json:"delta" validate:"oneof=-1 1"`
	}
	if err := core_http_request.DecodeAndValidateRequest(r, &body); err != nil {
		response.ErrorResponse(err, "Изменяйте навык по одному баллу")
		return
	}
	stat, err := h.gamificationService.AdjustStat(r.Context(), userID, statID, body.Delta)
	if err != nil {
		response.ErrorResponse(err, "Не удалось изменить навык")
		return
	}
	response.JSONResponse(statDTOFromDomain(stat), 200)
}

func (h *GamificationHTTPHandler) ListStats(w http.ResponseWriter, r *http.Request) {
	response := core_http_response.NewHTTPResponseHanlder(core_logger.FromContext(r.Context()), w)
	userID, ok := r.Context().Value(core_http_middleware.UserIDKey).(int)
	if !ok || userID <= 0 {
		response.ErrorResponse(core_errors.ErrUnauthorized, "Unauthorized")
		return
	}
	stats, err := h.gamificationService.ListStats(r.Context(), userID)
	if err != nil {
		response.ErrorResponse(err, "failed to list stats")
		return
	}
	result := make([]CreateStatResponse, 0, len(stats))
	for _, stat := range stats {
		result = append(result, statDTOFromDomain(stat))
	}
	response.JSONResponse(result, http.StatusOK)
}

func (h *GamificationHTTPHandler) DeleteStat(w http.ResponseWriter, r *http.Request) {
	response := core_http_response.NewHTTPResponseHanlder(core_logger.FromContext(r.Context()), w)
	userID, ok := r.Context().Value(core_http_middleware.UserIDKey).(int)
	if !ok || userID <= 0 {
		response.ErrorResponse(core_errors.ErrUnauthorized, "Unauthorized")
		return
	}
	statID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.ErrorResponse(core_errors.ErrInvalidArgument, "invalid stat ID")
		return
	}
	if err := h.gamificationService.DeleteStat(r.Context(), userID, statID); err != nil {
		response.ErrorResponse(err, "failed to delete stat")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
