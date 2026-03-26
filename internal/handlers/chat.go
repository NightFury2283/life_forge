package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"life_forge/internal/ai"
	"life_forge/internal/models"
	"life_forge/internal/storage"
	"life_forge/internal/usecases"
	"log"
	"net/http"
	"strings"
	"time"
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
	op := "internal/handlers/chat.go HandleChat"

	log.Println("New request /chat")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("%s: read body error: %v", op, err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	bodyReader := bytes.NewReader(bodyBytes)

	var user_question struct{ Text string }
	var message string

	contentType := r.Header.Get("Content-Type")
	log.Printf("Content-Type: %s", contentType)

	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(bodyReader).Decode(&user_question); err != nil {
			log.Printf("%s: decode json error: %v", op, err)
			http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
			return
		}
		message = user_question.Text
		log.Printf("JSON decode succesfully: Text = '%s'", message)
	} else {
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		if err := r.ParseForm(); err != nil {
			log.Printf("%s: parse form error: %v", op, err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		message = r.FormValue("message")
		if message == "" {
			message = r.FormValue("text")
		}
		log.Printf("Form decoded: message = '%s'", message)
	}

	if message == "" {
		log.Printf("%s: empty message", op)
		http.Error(w, `{"error": "No message provided"}`, http.StatusBadRequest)
		return
	}

	calendarData := ch.calendarStorage.GetCalendarPreview(r.Context(), 5)
	log.Printf("📅 Calendar data: %d symbols", len(calendarData))

	now := time.Now()
	timeContext := fmt.Sprintf("ВНИМАНИЕ! Сегодня: %s. Завтра: %s. Текущее время: %s. Все даты в JSON должны вычисляться относительно сегодня, используй часовой пояс +03:00 вместо Z!",
		now.Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		now.Format("15:04"))
	//запрос от пользователя (вместе с базовым промтом)
	promt_calendar := fmt.Sprintf("%s\n%s \n запрос от пользователя:%s\n %s", storage.PROMT_CALENDAR, calendarData, message, timeContext)

	response, err := makeRequestToAIGetResponse(r.Context(), ch, promt_calendar)
	if err != nil {
		log.Printf("%s: AI calendar error: %v", op, err)
		http.Error(w, `{"error": "AI service error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("Answer from AI: %d symbols", len(response))

	user_answer_calendar, events, err := usecases.ParseCalendarAIResponse(response)
	if err != nil {
		log.Printf("%s: parse calendar response error: %v", op, err)
	}

	log.Printf("Parsing events: %d events", len(events))
	saveEvents(ch, events)

	// ui
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		htmlResponse := fmt.Sprintf(`
            <div class="message p-4 rounded-2xl bg-gradient-to-r from-green-100 to-emerald-50 border border-green-200 max-w-3xl animate-slide-in mb-4">
                <div class="mb-1 font-semibold text-green-800">🤖 LifeForge AI:</div>
                <div class="text-gray-800">%s</div>
                %s
            </div>`,
			html.EscapeString(user_answer_calendar),
			formatEventsHTML(events))

		fmt.Fprint(w, htmlResponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	answ := user_answer_calendar
	if answ == "" {
		answ = "✅ Task completed!"
	}

	responseData := map[string]interface{}{
		"response":         answ,
		"events_count":     len(events),
		"calendar_preview": len(calendarData) > 50,
		"status":           "success",
	}

	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		log.Printf("%s: encode response error: %v", op, err)
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

func saveEvents(ch *ChatHandler, events []*models.EventRequest) {
	workers := make(chan struct{}, 5)

	for i, event := range events {

		workers <- struct{}{}

		go func(event *models.EventRequest) {
			defer func() {
				<-workers
			}()
			if event.IsEvent && event.Title != "" {
				log.Printf("Event %d: %s", i+1, event.Title)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				createdEvent, err := ch.calendarStorage.CreateEvent(ctx, *event)
				if err != nil {
					log.Printf("❌ Error to create event '%s': %v", event.Title, err)
				} else {
					log.Printf("✅ Event created: %s (ID: %s)", event.Title, createdEvent.Id)
				}

				err = ch.calendarStorage.SaveEvent(ctx, event)
				if err != nil {
					log.Printf("❌ Error to save event in db '%s': %v", event.Title, err)
				} else {
					log.Printf("✅ Event saved in db: %s", event.Title)
				}
			}
		}(event)
	}
}

// func mergeContexts(oldContext, newUpdates models.Context) models.Context {
// 	result := oldContext

// 	if len(newUpdates.Goals) > 0 {
// 		result.Goals = newUpdates.Goals
// 	}

// 	if len(newUpdates.Recent5) > 0 {
// 		result.Recent5 = newUpdates.Recent5
// 	}

// 	if len(newUpdates.Progress) > 0 {
// 		if result.Progress == nil {
// 			result.Progress = make(map[string]string)
// 		}
// 		for k, v := range newUpdates.Progress {
// 			result.Progress[k] = v
// 		}
// 	}

// 	return result
// }

func makeRequestToAIGetResponse(ctx context.Context, ch *ChatHandler, prompt string) (string, error) {
	op := "handlers.makeRequestToAIGetResponse"
	response, err := ch.aiClient.Generate(ctx, prompt)
	if err != nil {
		log.Printf("%s: AI error: %v", op, err)
		return "", err
	}
	return response, nil
}

// frontend
func formatEventsHTML(events []*models.EventRequest) string {
	if len(events) == 0 {
		return `<div class="mt-2 text-xs text-green-600">✅ Calendar checked, dont need new events</div>`
	}

	var htmlEvents strings.Builder
	htmlEvents.WriteString(`<div class="mt-3 pt-3 border-t border-green-200 space-y-2">`)
	htmlEvents.WriteString(`<div class="text-sm font-semibold text-green-700">📅 Created events:</div>`)

	for i, event := range events {
		if !event.IsEvent || event.Title == "" {
			continue
		}

		title := html.EscapeString(event.Title)
		timeStr := "сегодня"
		if event.StartTime != nil {
			timeStr = event.StartTime.Format("02.01 в 15:04")
		}

		duration := ""
		if event.DurationHours != nil {
			hours := *event.DurationHours
			if hours == 1 {
				duration = "1 час"
			} else if hours < 1 {
				duration = fmt.Sprintf("%.0f минут", hours*60)
			} else {
				duration = fmt.Sprintf("%.1f часа", hours)
			}
		}

		htmlEvents.WriteString(fmt.Sprintf(`
            <div class="p-2 bg-green-50 rounded-lg border border-green-200">
                <div class="font-medium text-green-800">%d. %s</div>
                <div class="text-xs text-green-600">⏰ %s • %s</div>
            </div>`, i+1, title, timeStr, duration))
	}

	htmlEvents.WriteString(`</div>`)
	return htmlEvents.String()
}
