package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"life_forge/internal/models"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type GoogleCalendarStorage struct {
	service *calendar.Service
	config  *oauth2.Config
}

// NewGoogleCalendarStorage инициализирует клиент.
// Если token.json есть - использует его.
// Если нет - возвращает ошибку (надо пройти Auth Flow).
func NewGoogleCalendarStorage() (*GoogleCalendarStorage, error) {
	data, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials.json: %w", err)
	}

	config, err := google.ConfigFromJSON(data, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Пытаемся загрузить сохраненный токен
	client, err := getClient(config)
	if err != nil {
		// Если токена нет, возвращаем объект, но без service
		// Это нормально! Мы инициализируем service после Auth Flow
		return &GoogleCalendarStorage{config: config}, nil
	}

	service, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar service: %w", err)
	}

	return &GoogleCalendarStorage{service: service, config: config}, nil
}

// IsAuthorized проверяет, есть ли валидный сервис
func (gcs *GoogleCalendarStorage) IsAuthorized() bool {
	return gcs.service != nil
}

// GetAuthURL возвращает ссылку для логина
func (gcs *GoogleCalendarStorage) GetAuthURL() string {
	return gcs.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// ExchangeCode меняет код от Google на токен и сохраняет его
func (gcs *GoogleCalendarStorage) ExchangeCode(code string) error {
	tok, err := gcs.config.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	saveToken("token.json", tok)

	client := gcs.config.Client(context.Background(), tok)
	service, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	gcs.service = service
	return nil
}

// Внутренние утилиты ==========================================

func getClient(config *oauth2.Config) (*http.Client, error) {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		return nil, err // Токена нет
	}
	return config.Client(context.Background(), tok), nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("Unable to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func (gcs *GoogleCalendarStorage) CreateEvent(event models.EventRequest) (*calendar.Event, error) { //(*calendar.Event, error)
	if event.StartTime == nil {
		log.Println("start time is required")
	}

	startTime := *event.StartTime
	var endTime time.Time

	if event.DurationHours != nil {
		endTime = startTime.Add(time.Duration(*event.DurationHours * float64(time.Hour)))
	} else {
		// Default: 1 hour
		endTime = startTime.Add(time.Hour)
	}

	googleEvent := &calendar.Event{
		Summary: event.Title,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "Europe/Moscow",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "Europe/Moscow",
		},
	}

	if event.Recurrence != nil {
		googleEvent.Recurrence = []string{fmt.Sprintf("RRULE:FREQ=%s", *event.Recurrence)}
	}

	if event.Description != nil {
		googleEvent.Description = *event.Description
	}

	return gcs.service.Events.Insert("primary", googleEvent).Do()
}

func (gcs *GoogleCalendarStorage) ListEvents(days int) ([]*calendar.Event, error) {
	if gcs.service == nil {
		return nil, fmt.Errorf("Календарь не подключен. Перейдите по /auth/google для авторизации.")
	}
	timeMin := time.Now().Format(time.RFC3339)
	timeMax := time.Now().AddDate(0, 0, days).Format(time.RFC3339)

	events, err := gcs.service.Events.List("primary").
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return events.Items, nil
}

func (gcs *GoogleCalendarStorage) GetCalendarPreview(days int) string {
	events, err := gcs.ListEvents(days)
	if err != nil {
		return fmt.Sprintf("Ошибка загрузки календаря: %v", err)
	}

	if len(events) == 0 {
		return "Нет запланированных событий"
	}

	var preview strings.Builder
	preview.WriteString("Ближайшие события:\n")

	for _, event := range events {
		start := event.Start.DateTime
		if start == "" {
			start = event.Start.Date
		}
		preview.WriteString(fmt.Sprintf("- %s: %s\n", start, event.Summary))
	}

	return preview.String()
}
