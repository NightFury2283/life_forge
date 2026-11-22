package calendar_repository_google

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	auth_google "github.com/NightFury2283/life_forge/internal/features/auth/repository/google"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type tokenStore struct{}

func (tokenStore) GetGoogleToken(_ context.Context, id int) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "user-" + strconv.Itoa(id), Expiry: time.Now().Add(time.Hour)}, nil
}
func (tokenStore) SaveGoogleToken(context.Context, int, *oauth2.Token) error { return nil }

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCreateIsIdempotentAndUserScoped(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, []byte(`{"web":{"client_id":"test","client_secret":"test","redirect_uris":["http://localhost/callback"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	provider, err := auth_google.NewGoogleAuthProvider(path, "")
	if err != nil {
		t.Fatal(err)
	}
	created := make(map[string]*calendar.Event)
	posts := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		status := 200
		var body []byte
		if r.Method == "POST" {
			posts++
			var event calendar.Event
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Fatal(err)
			}
			owner := r.Header.Get("Authorization")
			key := owner + "/" + event.Id
			if _, exists := created[key]; exists {
				status = 409
				body = []byte(`{"error":{"code":409,"message":"exists"}}`)
			} else {
				created[key] = &event
				body, _ = json.Marshal(event)
			}
		} else {
			id := filepath.Base(r.URL.Path)
			event := created[r.Header.Get("Authorization")+"/"+id]
			if event == nil {
				t.Fatal("looked up event for another user")
			}
			body, _ = json.Marshal(event)
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, client)
	repository := New(provider, tokenStore{})
	draft := domain.EventDraft{Title: "Test", Start: time.Now().Truncate(time.Second), Duration: 1, RequestID: "same-client-request"}
	first, err := repository.CreateEvent(ctx, 7, draft)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.CreateEvent(ctx, 7, draft)
	if err != nil {
		t.Fatal(err)
	}
	other, err := repository.CreateEvent(ctx, 8, draft)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.ID != second.ID || first.ID == other.ID || len(created) != 2 || posts != 3 {
		t.Fatalf("incorrect idempotency/isolation: ids=%q/%q/%q records=%d", first.ID, second.ID, other.ID, len(created))
	}
}
func TestFromGoogleAllDayAndMissingDates(t *testing.T) {
	t.Parallel()
	event, ok := fromGoogle("primary", &calendar.Event{Id: "a", Summary: "All day", Start: &calendar.EventDateTime{Date: "2026-11-24"}, End: &calendar.EventDateTime{Date: "2026-11-25"}})
	if !ok || !event.AllDay || event.End.Sub(event.Start) != 24*time.Hour {
		t.Fatalf("event=%+v ok=%v", event, ok)
	}
	if _, ok := fromGoogle("primary", &calendar.Event{}); ok {
		t.Fatal("missing dates accepted")
	}
}
