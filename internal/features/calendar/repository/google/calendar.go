package calendar_repository_google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	auth_google "github.com/NightFury2283/life_forge/internal/features/auth/repository/google"
	auth_service "github.com/NightFury2283/life_forge/internal/features/auth/service"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type Repository struct {
	provider *auth_google.GoogleAuthProvider
	tokens   auth_service.GoogleTokenStore
}

func New(provider *auth_google.GoogleAuthProvider, tokens auth_service.GoogleTokenStore) *Repository {
	return &Repository{provider, tokens}
}

type savingSource struct {
	base   oauth2.TokenSource
	store  auth_service.GoogleTokenStore
	ctx    context.Context
	userID int
	last   string
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		if err = s.store.SaveGoogleToken(s.ctx, s.userID, tok); err != nil {
			return nil, err
		}
		s.last = tok.AccessToken
	}
	return tok, nil
}
func (r *Repository) client(ctx context.Context, userID int) (*calendar.Service, error) {
	tok, err := r.tokens.GetGoogleToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	source := &savingSource{base: r.provider.TokenSource(ctx, tok), store: r.tokens, ctx: ctx, userID: userID, last: tok.AccessToken}
	client := oauth2.NewClient(ctx, source)
	client.Timeout = 20 * time.Second
	return calendar.NewService(ctx, option.WithHTTPClient(client))
}
func (r *Repository) ListCalendars(ctx context.Context, userID int) ([]domain.Calendar, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client, err := r.client(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Calendar, 0)
	err = client.CalendarList.List().MaxResults(250).Pages(ctx, func(page *calendar.CalendarList) error {
		for _, c := range page.Items {
			result = append(result, domain.Calendar{ID: c.Id, Summary: c.Summary, Color: c.BackgroundColor, Primary: c.Primary})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list Google calendars: %w", err)
	}
	return result, nil
}
func (r *Repository) ListEvents(ctx context.Context, userID int, start, end time.Time, ids []string) ([]domain.CalendarEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client, err := r.client(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.CalendarEvent, 0)
	for _, id := range ids {
		err = client.Events.List(id).TimeMin(start.Format(time.RFC3339)).TimeMax(end.Format(time.RFC3339)).SingleEvents(true).OrderBy("startTime").MaxResults(2500).Pages(ctx, func(page *calendar.Events) error {
			for _, e := range page.Items {
				event, ok := fromGoogle(id, e)
				if ok {
					result = append(result, event)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("list Google events: %w", err)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Start.Before(result[j].Start) })
	return result, nil
}
func (r *Repository) CreateEvent(ctx context.Context, userID int, d domain.EventDraft) (domain.CalendarEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client, err := r.client(ctx, userID)
	if err != nil {
		return domain.CalendarEvent{}, err
	}
	e := &calendar.Event{Summary: d.Title, Description: d.Description, Start: &calendar.EventDateTime{DateTime: d.Start.Format(time.RFC3339), TimeZone: "Europe/Moscow"}, End: &calendar.EventDateTime{DateTime: d.Start.Add(time.Duration(d.Duration * float64(time.Hour))).Format(time.RFC3339), TimeZone: "Europe/Moscow"}}
	if d.Recurrence != "" {
		e.Recurrence = []string{"RRULE:FREQ=" + strings.ToUpper(d.Recurrence)}
	}
	// Stable client request IDs prevent duplicate creation when the user retries.
	if d.RequestID != "" {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%d/%s", userID, d.RequestID)))
		e.Id = hex.EncodeToString(hash[:])
	}
	created, err := client.Events.Insert("primary", e).Context(ctx).Do()
	if err != nil {
		// The previous request may have succeeded before its response was lost.
		if e.Id != "" && isConflict(err) {
			created, err = client.Events.Get("primary", e.Id).Context(ctx).Do()
		}
		if err != nil {
			return domain.CalendarEvent{}, fmt.Errorf("create Google event: %w", err)
		}
	}
	result, _ := fromGoogle("primary", created)
	return result, nil
}
func fromGoogle(calendarID string, e *calendar.Event) (domain.CalendarEvent, bool) {
	if e.Start == nil || e.End == nil || e.Status == "cancelled" {
		return domain.CalendarEvent{}, false
	}
	parse := func(d *calendar.EventDateTime) (time.Time, error) {
		if d.DateTime != "" {
			return time.Parse(time.RFC3339, d.DateTime)
		}
		return time.Parse("2006-01-02", d.Date)
	}
	start, err := parse(e.Start)
	if err != nil {
		return domain.CalendarEvent{}, false
	}
	end, err := parse(e.End)
	if err != nil {
		return domain.CalendarEvent{}, false
	}
	return domain.CalendarEvent{ID: e.Id, CalendarID: calendarID, Title: e.Summary, Description: e.Description, Start: start, End: end, AllDay: e.Start.Date != ""}, true
}
func isConflict(err error) bool {
	var e *googleapi.Error
	return errors.As(err, &e) && e.Code == http.StatusConflict
}
