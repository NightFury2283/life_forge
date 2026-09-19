package users_transport_http

import (
	"net/http"

	core_http_middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
)

type ProfileResponse struct {
	Level          int `json:"level"`
	XP             int `json:"xp"`
	XPForNextLevel int `json:"xp_for_next_level"`
	Coins          int `json:"coins"`
}

func (h *UsersHTTPHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(core_http_middleware.UserIDKey).(int)

	if !ok || userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		//todo запись в логгер
		return
	}

	// 2. идём в service GetProfile

	// 3. отдаём ответ
	w.Header().Set("Content-Type", "application/json")
}
