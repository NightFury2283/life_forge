package users_transport_http

import (
	"net/http"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	core_http_middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
	core_http_response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
)

type ProfileResponse struct {
	Level          int `json:"level"`
	XP             int `json:"xp"`
	XPForNextLevel int `json:"xp_for_next_level"`
	Coins          int `json:"coins"`
}

func (h *UsersHTTPHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHanlder(log, w)

	userID, ok := ctx.Value(core_http_middleware.UserIDKey).(int)

	if !ok || userID == 0 {
		responseHandler.ErrorResponse(core_errors.ErrUnauthorized, "Unauthorized")
		return
	}

	log.Debug("Method GetProfile start")

	// 2. идём в service GetProfile
	userDomain, err := h.usersService.GetProfile(ctx, userID)
	if err != nil {
		responseHandler.ErrorResponse(err, "Get profile by handler from service")
		return
	}

	response := dtoFromDomain(userDomain)

	// 3. отдаём ответ
	responseHandler.JSONResponse(response, http.StatusOK)
}

func dtoFromDomain(progress *domain.UserProgress) ProfileResponse {
	return ProfileResponse{
		Level:          progress.Level,
		XP:             progress.XP,
		XPForNextLevel: progress.XPForNextLevel,
		Coins:          progress.Coins,
	}
}
