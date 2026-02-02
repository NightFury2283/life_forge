package handlers

import (
	"encoding/json"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"log"
	"net/http"
	"strconv"
)

type JournalHandler struct {
	storage *storage.JournalStorage
}

func NewJournalHandler(s *storage.JournalStorage) *JournalHandler {
	return &JournalHandler{storage: s}
}

func (jh *JournalHandler) HandleCreateEntry(w http.ResponseWriter, r *http.Request) {
	op := "internal/handlers/journal.go HandleCreateEntry"
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		log.Println("Method not Allowed ", r.Method, " in ", op)
		return
	}

	var entry models.JournalEntry

	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Couldnt decode json. Wrong request.", http.StatusBadRequest)
		log.Println("Couldnt decode json. Wrong request ", " in ", op)
		return
	}

	//если всё ок

	if err := jh.storage.CreateEntry(r.Context(), &entry); err != nil {
		http.Error(w, "Couldnt create entry with error", http.StatusConflict)
		log.Println("Couldnt create entry with error ", err, " in ", op)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]string{
		"status":  "created",
		"message": "Entry created successfully",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("Failed to encode response in ", op, "with error: ", err)
		return
	}
}

func (jh *JournalHandler) HandleGetEntries(w http.ResponseWriter, r *http.Request) {
	op := "internal/handlers/journal.go HandleGetEntries"

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		log.Println("Method not Allowed ", r.Method, " in ", op)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err != nil {
			limit = l
		}
	}

	entries, err := jh.storage.GetEntries(r.Context(), limit)

	if err != nil {
		http.Error(w, "Couldnt get entries.", http.StatusConflict)
		log.Println("Couldnt get entries", " in ", op)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "success",
		"data":   entries,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("Failed to encode response in ", op, "with error: ", err)
		return
	}
}
