package handlers

import (
	"life_forge/internal/storage"
	"net/http"
)

type AuthHandler struct {
	calendarStorage *storage.YandexCalendarStorage
}

func NewAuthHandler(cs *storage.YandexCalendarStorage) *AuthHandler {
	return &AuthHandler{calendarStorage: cs}
}

// /auth/google -> redirect to google
func (h *AuthHandler) HandleYandexLogin(w http.ResponseWriter, r *http.Request) {
	url := h.calendarStorage.GetAuthURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// /auth/callback -> Google send code here
func (h *AuthHandler) HandleYandexCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	err := h.calendarStorage.ExchangeCode(code)
	if err != nil {
		http.Error(w, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
}
