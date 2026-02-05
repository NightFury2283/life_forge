package handlers

import (
	"encoding/json"
	"fmt"
	"life_forge/internal/ai"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"life_forge/internal/usecases"
	"log"
	"net/http"
	"strings"
)

type ChatHandler struct {
	contextStorage *storage.ContextStorage
	aiClient       *ai.GigaChatClient
}

func NewChatHandler(contextStorage *storage.ContextStorage, aiClient *ai.GigaChatClient) *ChatHandler {
	return &ChatHandler{
		contextStorage: contextStorage,
		aiClient:       aiClient,
	}
}

func (ch *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	op := "handlers.HandleChat"

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//text from user
	var input struct{ Text string }
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("%s: decode error: %v", op, err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	//current context
	context, err := ch.contextStorage.GetContextByID(r.Context(), 1)
	if err != nil {
		log.Printf("%s: get context error: %v", op, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	//calendar
	// TODO: add calendar
	calendarData := ch.contextStorage.GetCalendarPreview(r.Context())

	prompt := fmt.Sprintf("%s\n\nЦели: %s\nНедавние действия: %s\nПрогресс: %v\nКалендарь: %s\n\nЗапрос пользователя:\n%s",
		storage.PROMPT,
		strings.Join(context.Goals, ", "),
		strings.Join(context.Recent5, "; "),
		context.Progress,
		calendarData,
		input.Text)

	//get answer from ai
	response, err := ch.aiClient.Generate(prompt)
	if err != nil {
		log.Printf("%s: AI error: %v", op, err)
		http.Error(w, "AI service error", http.StatusInternalServerError)
		return
	}

	userAnswer, aiUpdates := usecases.ParseAIResponse(response)

	// add updates to old context in db
	mergedContext := mergeContexts(context, aiUpdates)

	if err := ch.contextStorage.SaveContext(r.Context(), &mergedContext); err != nil {
		log.Printf("%s: save context error: %v", op, err)
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]string{
		"response":      userAnswer,
		"full_response": response,
	}); err != nil {
		log.Printf("%s: encode response error: %v", op, err)
	}
}

func mergeContexts(oldContext, newUpdates models.Context) models.Context {
	result := oldContext

	if len(newUpdates.Goals) > 0 {
		result.Goals = newUpdates.Goals
	}

	if len(newUpdates.Recent5) > 0 {
		result.Recent5 = newUpdates.Recent5
	}

	if len(newUpdates.Progress) > 0 {
		if result.Progress == nil {
			result.Progress = make(map[string]string)
		}
		for k, v := range newUpdates.Progress {
			result.Progress[k] = v
		}
	}

	return result
}
