package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"life_forge/internal/models"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

type YandexCalendarStorage struct {
	client     *caldav.Client
	calendar   *caldav.Calendar
	config     *oauth2.Config
	pool       *pgxpool.Pool
	httpClient *http.Client
}

func NewYandexCalendarStorage(pool *pgxpool.Pool, clientID string, clientSecret string) (*YandexCalendarStorage, error) {
	config := &oauth2.Config{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://oauth.yandex.ru/authorize",
			TokenURL:  "https://oauth.yandex.ru/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		RedirectURL: "http://localhost:8080/auth/callback",
	}

	storage := &YandexCalendarStorage{
		config: config,
		pool:   pool,
	}

	httpClient, err := getClient(config)
	if err == nil {
		storage.httpClient = httpClient
		err = storage.initCalDAVClient()
		if err != nil {
			log.Printf("Failed to init CalDAV (maybe token expired or invalid): %v", err)
		}
	}

	return storage, nil
}

func (ycs *YandexCalendarStorage) initCalDAVClient() error {
	client, err := caldav.NewClient(ycs.httpClient, "https://caldav.yandex.ru/")
	if err != nil {
		return err
	}

	principal, err := client.FindCurrentUserPrincipal(context.Background())
	if err != nil {
		return err
	}

	homeSet, err := client.FindCalendarHomeSet(context.Background(), principal)
	if err != nil {
		return err
	}

	calendars, err := client.FindCalendars(context.Background(), homeSet)
	if err != nil || len(calendars) == 0 {
		return fmt.Errorf("no calendars found or err: %v", err)
	}

	// For now, grab the first available calendar (often "events-xxx")
	ycs.client = client
	ycs.calendar = &calendars[0]
	return nil
}

func (ycs *YandexCalendarStorage) IsAuthorized() bool {
	return ycs.client != nil
}

func (ycs *YandexCalendarStorage) GetAuthURL() string {
	return ycs.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (ycs *YandexCalendarStorage) ExchangeCode(code string) error {
	tok, err := ycs.config.Exchange(context.Background(), code)
	if err != nil {
		if rErr, ok := err.(*oauth2.RetrieveError); ok {
			return fmt.Errorf("failed to exchange code (Yandex): HTTP %d %s", rErr.Response.StatusCode, string(rErr.Body))
		}
		return fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	tok.TokenType = "OAuth"
	saveToken("token.json", tok)

	client := ycs.config.Client(context.Background(), tok)
	client.Transport = &yandexETagFixTransport{Base: client.Transport}
	ycs.httpClient = client
	
	return ycs.initCalDAVClient()
}

func getClient(config *oauth2.Config) (*http.Client, error) {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		return nil, err
	}
	tok.TokenType = "OAuth"
	client := config.Client(context.Background(), tok)
	client.Transport = &yandexETagFixTransport{Base: client.Transport}
	return client, nil
}

type yandexETagFixTransport struct {
	Base http.RoundTripper
}

func (t *yandexETagFixTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 207 || resp.StatusCode == 200 {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "xml") {
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				resp.Body.Close()
				re := regexp.MustCompile(`(<(?:[a-zA-Z0-9\-]+:)?getetag[^>]*>)\s*([^"<]+?)\s*(</(?:[a-zA-Z0-9\-]+:)?getetag>)`)
				fixedBody := re.ReplaceAll(body, []byte(`${1}"${2}"${3}`))
				resp.Body = io.NopCloser(bytes.NewReader(fixedBody))
				resp.ContentLength = int64(len(fixedBody))
				resp.Header.Set("Content-Length", strconv.Itoa(len(fixedBody)))
			}
		}
	}
	return resp, nil
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

func (ycs *YandexCalendarStorage) CreateEvent(ctx context.Context, eventReq models.EventRequest) (*calendar.Event, error) {
	if !ycs.IsAuthorized() || ycs.calendar == nil {
		return nil, fmt.Errorf("Календарь Яндекса не подключен. Сначала авторизуйтесь (Яндекс Auth).")
	}

	if eventReq.StartTime == nil {
		log.Println("start time is required")
		return nil, fmt.Errorf("start time required")
	}

	startTime := *eventReq.StartTime
	var endTime time.Time

	if eventReq.DurationHours != nil {
		endTime = startTime.Add(time.Duration(*eventReq.DurationHours * float64(time.Hour)))
	} else {
		endTime = startTime.Add(time.Hour)
	}

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//LifeForge AI//ru")

	event := ical.NewEvent()
	uid := fmt.Sprintf("%d-life-forge-ai@localhost", time.Now().UnixNano())
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())
	event.Props.SetText(ical.PropSummary, eventReq.Title)
	event.Props.SetDateTime(ical.PropDateTimeStart, startTime.UTC())
	event.Props.SetDateTime(ical.PropDateTimeEnd, endTime.UTC())

	if eventReq.Description != nil {
		event.Props.SetText(ical.PropDescription, *eventReq.Description)
	}

	if eventReq.Recurrence != nil && *eventReq.Recurrence != "" {
		recurrenceRule := formatRecurrenceRule(*eventReq.Recurrence)
		if recurrenceRule != "" {
			rruleVal := strings.TrimPrefix(recurrenceRule, "RRULE:")
			event.Props.SetText(ical.PropRecurrenceRule, rruleVal)
		}
	}

	cal.Children = append(cal.Children, event.Component)

	href := fmt.Sprintf("%s%s.ics", ycs.calendar.Path, uid)
	_, err := ycs.client.PutCalendarObject(ctx, href, cal)
	if err != nil {
		return nil, fmt.Errorf("failed to save to calendar: %w", err)
	}

	// Returning Google Calendar struct to keep logic intact in handlers
	return &calendar.Event{
		Id:      uid,
		Summary: eventReq.Title,
	}, nil
}

func formatRecurrenceRule(recurrence string) string {
	switch strings.ToUpper(recurrence) {
	case "DAILY": return "FREQ=DAILY"
	case "WEEKLY": return "FREQ=WEEKLY"
	case "MONTHLY": return "FREQ=MONTHLY"
	case "YEARLY": return "FREQ=YEARLY"
	default:
		if strings.HasPrefix(strings.ToUpper(recurrence), "RRULE:") {
			return strings.TrimPrefix(strings.ToUpper(recurrence), "RRULE:")
		}
		return fmt.Sprintf("FREQ=%s", strings.ToUpper(recurrence))
	}
}

func (ycs *YandexCalendarStorage) SaveEvent(ctx context.Context, event *models.EventRequest) error {
	op := "internal/storage/yandex_calendar.go SaveEvent"

	sql_query := `INSERT INTO events (is_event, title, start_time, duration_hours, recurrence, description) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := ycs.pool.Exec(ctx, sql_query, event.IsEvent, event.Title, event.StartTime, event.DurationHours, event.Recurrence, event.Description)

	if err != nil {
		return fmt.Errorf("%s: failed to save context data: %w", op, err)
	}
	return nil
}

func (ycs *YandexCalendarStorage) ListEvents(ctx context.Context, timeMin, timeMax time.Time) ([]*calendar.Event, error) {
	if ycs.client == nil || ycs.calendar == nil {
		return nil, fmt.Errorf("Календарь Яндекса не подключен. Авторизуйтесь.")
	}

	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{
				Name: "VEVENT",
			}},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name: "VEVENT",
				Start: timeMin,
				End:   timeMax,
			}},
		},
	}

	objects, err := ycs.client.QueryCalendar(ctx, ycs.calendar.Path, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var results []*calendar.Event
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for _, event := range obj.Data.Events() {
			start, limitErr := event.DateTimeStart(time.Local)
			if limitErr != nil {
				continue
			}
			end, _ := event.DateTimeEnd(time.Local)
			summary, _ := event.Props.Text(ical.PropSummary)

			gEvent := &calendar.Event{
				Id:      obj.Path,
				Summary: summary,
				Start: &calendar.EventDateTime{
					DateTime: start.Format(time.RFC3339),
				},
				End: &calendar.EventDateTime{
					DateTime: end.Format(time.RFC3339),
				},
			}
			results = append(results, gEvent)
		}
	}

	return results, nil
}

func (ycs *YandexCalendarStorage) GetCalendarPreview(ctx context.Context, days int) string {
	timeMin := time.Now().UTC()
	timeMax := time.Now().AddDate(0, 0, days).UTC()
	events, err := ycs.ListEvents(ctx, timeMin, timeMax)
	if err != nil {
		return fmt.Sprintf("Error to load calendar: %v", err)
	}

	if len(events) == 0 {
		return "No events"
	}

	var preview strings.Builder
	preview.WriteString("Closest events:\n")

	for _, event := range events {
		start := event.Start.DateTime
		if start == "" {
			start = event.Start.Date
		}
		preview.WriteString(fmt.Sprintf("- %s: %s\n", start, event.Summary))
	}

	return preview.String()
}
