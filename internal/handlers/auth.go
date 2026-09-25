package handlers

import (
	"github.com/NightFury2283/life_forge/internal/storage"
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

