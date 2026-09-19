package handlers

import (
	"encoding/json"
	"github.com/NightFury2283/life_forge/internal/storage"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type GamificationHandler struct {
	userStorage     *storage.UserStorage
	calendarStorage *storage.GoogleCalendarStorage
}

func NewGamificationHandler(us *storage.UserStorage, cs *storage.GoogleCalendarStorage) *GamificationHandler {
	return &GamificationHandler{
		userStorage:     us,
		calendarStorage: cs,
	}
}

//получает текущего пользователя из Google ID
func (h *GamificationHandler) getCurrentUserID(r *http.Request) (int, error) {
	// Получаем Google ID из авторизованного календаря
	googleID, err := h.calendarStorage.GetGoogleUserID(r.Context())
	if err != nil {
		return 0, err
	}

	// Находим пользователя в БД
	user, err := h.userStorage.GetOrCreateUserByGoogleID(r.Context(), googleID, "", "")
	if err != nil {
		return 0, err
	}

	return user.ID, nil
}

// GET /api/profile - получить профиль пользователя
func (h *GamificationHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getCurrentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	progress, err := h.userStorage.GetProgress(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get progress", http.StatusInternalServerError)
		return
	}

	stats, err := h.userStorage.GetStats(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"level":       progress.Level,
		"xp":          progress.XP,
		"xp_for_next": progress.XPForNextLevel,
		"stats":       stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /api/stats - создать характеристику
func (h *GamificationHandler) HandleCreateStat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userID, err := h.getCurrentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stat, err := h.userStorage.CreateStat(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		http.Error(w, "Failed to create stat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stat)
}

// PUT /api/stats/{id}?delta=1 - обновить характеристику
func (h *GamificationHandler) HandleUpdateStat(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	statID, err := strconv.Atoi(pathParts[3])
	if err != nil {
		http.Error(w, "Invalid stat ID", http.StatusBadRequest)
		return
	}

	deltaStr := r.URL.Query().Get("delta")
	delta, err := strconv.Atoi(deltaStr)
	if err != nil || (delta != 1 && delta != -1) {
		http.Error(w, "Delta must be 1 or -1", http.StatusBadRequest)
		return
	}

	if err := h.userStorage.UpdateStatValue(r.Context(), statID, delta); err != nil {
		http.Error(w, "Failed to update stat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DELETE /api/stats/{id} - удалить характеристику
func (h *GamificationHandler) HandleDeleteStat(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	statID, err := strconv.Atoi(pathParts[3])
	if err != nil {
		http.Error(w, "Invalid stat ID", http.StatusBadRequest)
		return
	}

	if err := h.userStorage.DeleteStat(r.Context(), statID); err != nil {
		http.Error(w, "Failed to delete stat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GET /api/tasks - получить задачи из календаря (невыполненные)
func (h *GamificationHandler) HandleGetTasks(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getCurrentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем события из календаря на ближайшие 14 дней
	timeMin := time.Now().UTC()
	timeMax := time.Now().AddDate(0, 0, 14).UTC()

	events, err := h.calendarStorage.ListEvents(r.Context(), timeMin, timeMax)
	if err != nil {
		http.Error(w, "Failed to get calendar events", http.StatusInternalServerError)
		return
	}

	// Фильтруем невыполненные
	var tasks []map[string]interface{}
	for _, event := range events {
		completed, _ := h.userStorage.IsTaskCompleted(r.Context(), userID, event.Id)
		if !completed && event.Summary != "" {
			tasks = append(tasks, map[string]interface{}{
				"id":         event.Id,
				"title":      event.Summary,
				"start_time": event.Start.DateTime,
				"xp_reward":  10,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// POST /api/tasks/complete - отметить задачу выполненной
func (h *GamificationHandler) HandleCompleteTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventID    string `json:"event_id"`
		EventTitle string `json:"event_title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userID, err := h.getCurrentUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Проверяем, не выполнена ли уже
	completed, _ := h.userStorage.IsTaskCompleted(r.Context(), userID, req.EventID)
	if completed {
		http.Error(w, "Task already completed", http.StatusBadRequest)
		return
	}

	// Выполняем задачу
	xpGained := 10
	if err := h.userStorage.CompleteTask(r.Context(), userID, req.EventID, req.EventTitle, xpGained); err != nil {
		log.Printf("Error completing task: %v", err)
		http.Error(w, "Failed to complete task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"xp_gained": xpGained,
	})
}
