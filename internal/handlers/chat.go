package handlers

import (
	"encoding/json"
	"fmt"
	"html"
	"life_forge/internal/ai"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"life_forge/internal/usecases"
	"log"
	"net/http"
	"strings"
)

type ChatHandler struct {
	contextStorage  *storage.ContextStorage
	aiClient        *ai.GigaChatClient
	calendarStorage *storage.GoogleCalendarStorage
}

func NewChatHandler(contextStorage *storage.ContextStorage, aiClient *ai.GigaChatClient, calendarStorage *storage.GoogleCalendarStorage) *ChatHandler {
	return &ChatHandler{
		contextStorage:  contextStorage,
		aiClient:        aiClient,
		calendarStorage: calendarStorage,
	}
}

func (ch *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	op := "handlers.chat.go HandleChat"

	//check method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//text from user - поддержка JSON и form-urlencoded для HTMX
	var user_question struct{ Text string }
	var message string

	// Проверяем Content-Type для HTMX (form) и JS fetch (json)
	if r.Header.Get("Content-Type") == "application/json" || strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&user_question); err != nil {
			log.Printf("%s: decode json error: %v", op, err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		message = user_question.Text
	} else {
		// HTMX отправляет form-urlencoded
		if err := r.ParseForm(); err != nil {
			log.Printf("%s: parse form error: %v", op, err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		message = r.FormValue("message")
		if message == "" {
			message = r.FormValue("text")
		}
	}

	if message == "" {
		http.Error(w, "No message", http.StatusBadRequest)
		return
	}

	//current curr_context
	/*
		curr_context, err := ch.contextStorage.GetContextByID(r.Context(), 1)
		if err != nil {
			log.Printf("%s: get context error: %v", op, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	*/

	//one step. Promt for calendar (send future 5 days of calendar), receive new events and answer to user.

	//calendar

	//show next 5 days of Calendar to AI
	calendarData := ch.calendarStorage.GetCalendarPreview(5)

	promt_calendar := fmt.Sprintf("%s\n%s ", storage.PROMT_CALENDAR, calendarData)

	response, err := makeRequestToAIGetResponse(ch, promt_calendar)
	if err != nil {
		log.Printf("%s: AI calendar error: %v", op, err)
		http.Error(w, "AI service error", http.StatusInternalServerError)
		return
	}

	temp_response_calendar := response

	user_answer_calendar, events, err := usecases.ParseCalendarAIResponse(response)
	if err != nil {
		log.Printf("%s: parse calendar response error: %v", op, err)
	}

	for _, event := range events {
		ch.calendarStorage.CreateEvent(*event)
	}
	/*
		prompt_db := fmt.Sprintf("%s\n\nЦели: %s\nНедавние действия: %s\nПрогресс: %v\nКалендарь: %s\n\nЗапрос пользователя:\n%s",
			storage.PROMT_DB,
			strings.Join(curr_context.Goals, ", "),
			strings.Join(curr_context.Recent5, "; "),
			curr_context.Progress,
			calendarData,
			message)

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
	*/

	// check htmx query - return html fragment
	if r.Header.Get("HX-Request") != "" || r.Header.Get("HX-Trigger") != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		htmlResponse := fmt.Sprintf(`
            <div class="message p-4 rounded-2xl bg-gradient-to-r from-indigo-500 to-purple-600 text-white shadow-xl max-w-3xl animate-slide-in mb-4">
                <div class="mb-2 text-indigo-100">🤖 LifeForge AI</div>
                <div>%s</div>
                %s
            </div>`,
			html.EscapeString(user_answer_calendar+"\n"), //html.EscapeString(user_answer_calendar+"\n\n"+userAnswer),
			formatEventsHTML(events))

		fmt.Fprint(w, htmlResponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	answ := user_answer_calendar

	if err := json.NewEncoder(w).Encode(map[string]string{
		//TODO: delete calendar response
		"response_calendar": temp_response_calendar,
		"response":          answ,
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

// show calendar events as nice HTML for UI
func formatEventsHTML(events []*models.EventRequest) string {
	if len(events) == 0 {
		return ""
	}

	var htmlEvents strings.Builder
	htmlEvents.WriteString("<div class='mt-3 pt-3 border-t border-indigo-200 space-y-2'>")
	htmlEvents.WriteString("<div class='text-xs font-semibold text-indigo-300 uppercase tracking-wide'>📅 Новые события:</div>")

	for _, event := range events {
		title := html.EscapeString(event.Title)
		timeStr := event.StartTime.Format("02.01 15:04")
		duration := ""
		if event.DurationHours != nil {
			duration = fmt.Sprintf("%.1fч", *event.DurationHours)
		}
		htmlEvents.WriteString(fmt.Sprintf(`
            <div class="p-3 bg-white/30 backdrop-blur-sm rounded-xl border border-indigo-200 text-sm">
                <div class="font-semibold">%s</div>
                <div class="text-indigo-100">%s %s</div>
            </div>`, title, timeStr, duration))
	}

	htmlEvents.WriteString("</div>")
	return htmlEvents.String()
}
