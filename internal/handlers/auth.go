package handlers

import (
	"life_forge/internal/storage"
	"log"
	"net/http"
)

type AuthHandler struct {
	calendarStorage *storage.GoogleCalendarStorage
	userStorage     *storage.UserStorage
}

func NewAuthHandler(
	cs *storage.GoogleCalendarStorage,
	us *storage.UserStorage,
) *AuthHandler {
	return &AuthHandler{
		calendarStorage: cs,
		userStorage:     us,
	}
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

	// Обмениваем код на токен
	err := h.calendarStorage.ExchangeCode(code)
	if err != nil {
		http.Error(w, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем Google ID пользователя
	googleID, err := h.calendarStorage.GetGoogleUserID(r.Context())
	if err != nil {
		log.Printf("⚠️ Warning: failed to get google id: %v", err)
		http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
		return
	}

	// TODO: Получить email и имя пользователя (пока заглушка)
	email := googleID + "@gmail.com"
	name := "User_" + googleID[:8]

	// Создаём или получаем пользователя в БД
	user, err := h.userStorage.GetOrCreateUserByGoogleID(r.Context(), googleID, email, name)
	if err != nil {
		log.Printf("❌ Error creating user: %v", err)
		http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
		return
	}

	log.Printf("✅ User logged in: %s (ID: %d, Google ID: %s)", user.Name, user.ID, user.GoogleID)

	// TODO: Сохранить user.ID в сессию или JWT токен
	// Пока просто редиректим

	http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
}
