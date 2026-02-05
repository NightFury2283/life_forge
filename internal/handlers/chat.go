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

	//check method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//text from user
	var user_question struct{ Text string }
	if err := json.NewDecoder(r.Body).Decode(&user_question); err != nil {
		log.Printf("%s: decode error: %v", op, err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	//current curr_context
	curr_context, err := ch.contextStorage.GetContextByID(r.Context(), 1)
	if err != nil {
		log.Printf("%s: get context error: %v", op, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	//Two steps. Promt for calendar if needed, then promt for db update and answer to user

	//promt_calendar send request to ai to check if user wants to create calendar event
	//promt_db send request to ai to get answer for user and updates for db

	//calendar
	
	// TODO: add calendar. Load real calendar data from storage
	calendarData := ch.contextStorage.GetCalendarPreview(r.Context())

	promt_calendar := fmt.Sprintf("%s\n%s ", storage.PROMT_CALENDAR, calendarData)

	response, err := makeRequestToAIGetResponse(ch, promt_calendar)
	if err != nil {
		log.Printf("%s: AI calendar error: %v", op, err)
		http.Error(w, "AI service error", http.StatusInternalServerError)
		return
	}

	// TODO: use events _ for google calendar integration
	user_answer_calendar, _, err := usecases.ParseCalendarAIResponse(response)
	if err != nil {
		log.Printf("%s: parse calendar response error: %v", op, err)
		http.Error(w, "AI response parsing error", http.StatusInternalServerError)
		return
	}

	prompt_db := fmt.Sprintf("%s\n\nЦели: %s\nНедавние действия: %s\nПрогресс: %v\nКалендарь: %s\n\nЗапрос пользователя:\n%s",
		storage.PROMT_DB,
		strings.Join(curr_context.Goals, ", "),
		strings.Join(curr_context.Recent5, "; "),
		curr_context.Progress,
		calendarData,
		user_question.Text)

	response, err = makeRequestToAIGetResponse(ch, prompt_db)
	if err != nil {
		log.Printf("%s: AI DB error: %v", op, err)
		http.Error(w, "AI service error", http.StatusInternalServerError)
		return
	}

	userAnswer, aiUpdates := usecases.ParseAIResponse(response)

	// add updates to old context in db
	mergedContext := mergeContexts(curr_context, aiUpdates)

	if err := ch.contextStorage.SaveContext(r.Context(), &mergedContext); err != nil {
		log.Printf("%s: save context error: %v", op, err)
	}

	w.Header().Set("Content-Type", "application/json")

	answ := user_answer_calendar + "\n\n" + userAnswer

	if err := json.NewEncoder(w).Encode(map[string]string{
		"response": answ,
		//TODO: delete full response from answer to user. It is only for debug now
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

func makeRequestToAIGetResponse(ch *ChatHandler, prompt string) (string, error) {
	op := "handlers.makeRequestToAIGetResponse"
	//get answer from ai
	response, err := ch.aiClient.Generate(prompt)
	if err != nil {
		log.Printf("%s: AI error: %v", op, err)
		return "", err
	}
	return response, nil
}
