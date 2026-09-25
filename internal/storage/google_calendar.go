package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/NightFury2283/life_forge/internal/models"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	whereSaveEvent = "primary"
)



type GoogleCalendarStorage struct {
	service    *calendar.Service
	config     *oauth2.Config
	pool       *pgxpool.Pool
	httpClient *http.Client
	userInfo   *GoogleUserInfo // Добавляем кэш
}

func NewGoogleCalendarStorage(pool *pgxpool.Pool) (*GoogleCalendarStorage, error) {
	data, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials.json: %w", err)
	}

	config, err := google.ConfigFromJSON(data, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	client, err := getClient(config)
	if err != nil {
		return &GoogleCalendarStorage{config: config, pool: pool}, nil
	}

	service, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar service: %w", err)
	}

	return &GoogleCalendarStorage{
		service:    service,
		config:     config,
		pool:       pool,
		httpClient: client,
	}, nil
}

func (gcs *GoogleCalendarStorage) IsAuthorized() bool {
	return gcs.service != nil && gcs.httpClient != nil
}





//==========================================

func getClient(config *oauth2.Config) (*http.Client, error) {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		return nil, err
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





func formatRecurrenceRule(recurrence string) string {
	switch strings.ToUpper(recurrence) {
	case "DAILY":
		return "RRULE:FREQ=DAILY"
	case "WEEKLY":
		return "RRULE:FREQ=WEEKLY"
	case "MONTHLY":
		return "RRULE:FREQ=MONTHLY"
	case "YEARLY":
		return "RRULE:FREQ=YEARLY"
	default:
		if strings.HasPrefix(strings.ToUpper(recurrence), "RRULE:") {
			return recurrence
		}
		return fmt.Sprintf("RRULE:FREQ=%s", strings.ToUpper(recurrence))
	}
}

func (gcs *GoogleCalendarStorage) SaveEventInDB(ctx context.Context, event *models.EventRequest) error {
	op := "internal/storage/google_calendar.go SaveEvent"

	sql_query := `
	INSERT INTO events (is_event, title, start_time, duration_hours, recurrence, description) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := gcs.pool.Exec(ctx, sql_query,
		event.IsEvent,
		event.Title,
		event.StartTime,
		event.DurationHours,
		event.Recurrence,
		event.Description,
	)

	if err != nil {
		log.Println("Error with Exec method in ", op, " with error: ", err)
		return fmt.Errorf("%s: failed to save context data: %w", op, err)
	}

	return nil
}






