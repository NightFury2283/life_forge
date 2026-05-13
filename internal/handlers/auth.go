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

func NewAuthHandler(cs *storage.GoogleCalendarStorage, us *storage.UserStorage) *AuthHandler {
	return &AuthHandler{
		calendarStorage: cs,
		userStorage:     us,
	}
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.calendarStorage.GetAuthURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	log.Printf("Received auth code, length: %d", len(code))

	// Обмениваем код на токен
	err := h.calendarStorage.ExchangeCode(code)
	if err != nil {
		log.Printf("ExchangeCode error: %v", err)
		http.Error(w, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Token exchanged successfully")

	// Получаем информацию о пользователе
	userInfo, err := h.calendarStorage.GetGoogleUserInfo(r.Context())
	if err != nil {
		log.Printf("⚠️ Warning: failed to get user info: %v", err)
		http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
		return
	}

	googleID := userInfo.ID
	email := userInfo.Email
	name := userInfo.Name

	log.Printf("✅ User info - ID: %s, Email: %s, Name: %s", googleID, email, name)

	if googleID == "" {
		log.Printf("⚠️ Warning: google ID is empty")
		http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
		return
	}

	// Создаём или получаем пользователя в БД
	user, err := h.userStorage.GetOrCreateUserByGoogleID(r.Context(), googleID, email, name)
	if err != nil {
		log.Printf("❌ Error creating user: %v", err)
		http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
		return
	}

	log.Printf("🎉 User logged in: %s (ID: %d)", user.Name, user.ID)

	http.Redirect(w, r, "http://localhost:8080", http.StatusPermanentRedirect)
}
