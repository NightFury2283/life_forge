package handlers

import (
	"fmt"
	"life_forge/internal/storage"
	"net/http"
)

type AuthHandler struct {
	calendarStorage *storage.GoogleCalendarStorage
}

func NewAuthHandler(cs *storage.GoogleCalendarStorage) *AuthHandler {
	return &AuthHandler{calendarStorage: cs}
}

// /auth/google -> redirect to google
func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.calendarStorage.GetAuthURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// /auth/callback -> Google send code here
func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
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

	fmt.Fprintf(w, "Авторизация успешна! Можете закрыть окно и вернуться в чат.")
}
