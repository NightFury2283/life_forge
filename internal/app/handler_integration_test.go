package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	calendar_service "github.com/NightFury2283/life_forge/internal/features/calendar/service"
	chat_service "github.com/NightFury2283/life_forge/internal/features/chat/service"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NightFury2283/life_forge/internal/app"
	tokens "github.com/NightFury2283/life_forge/internal/core/auth/jwt"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	"github.com/NightFury2283/life_forge/internal/demo"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Only Google is replaced: HTTP, OAuth state handling, JWT, services and memory
// storage are real. These tests deliberately do not claim PostgreSQL coverage.
type googleProvider struct{}

func (googleProvider) GetAuthURL(state string) string {
	return "https://accounts.example/authorize?state=" + url.QueryEscape(state)
}
func (googleProvider) ExchangeCode(_ context.Context, code string) (*oauth2.Token, error) {
	if code == "denied" {
		return nil, fmt.Errorf("provider rejected code")
	}
	return &oauth2.Token{AccessToken: code}, nil
}
func (googleProvider) GetGoogleUserInfo(_ context.Context, tok *oauth2.Token) (*domain.GoogleUserInfo, error) {
	return &domain.GoogleUserInfo{ID: tok.AccessToken, Name: "Test user", Email: tok.AccessToken + "@example.com"}, nil
}

func startApp(t *testing.T, demoMode bool, extras ...app.Options) *httptest.Server {
	t.Helper()
	repo := demo.NewRepository()
	tokens := tokens.NewTokenService(tokens.JWTConfig{SecretKey: "integration-secret-at-least-thirty-two-bytes"})
	options := app.Options{Demo: demoMode}
	if len(extras) > 0 {
		options = extras[0]
		options.Demo = demoMode
	}
	server := httptest.NewServer(app.NewHandler(repo, repo, googleProvider{}, tokens, &core_logger.Logger{Logger: zap.NewNop()}, options))
	t.Cleanup(server.Close)
	return server
}

type plannerRepository struct {
	mu     sync.Mutex
	events map[int][]domain.CalendarEvent
	writes int
}

func (p *plannerRepository) ListCalendars(context.Context, int) ([]domain.Calendar, error) {
	return []domain.Calendar{{ID: "primary", Summary: "My calendar", Primary: true}}, nil
}
func (p *plannerRepository) ListEvents(_ context.Context, id int, _, _ time.Time, _ []string) ([]domain.CalendarEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := append([]domain.CalendarEvent{}, p.events[id]...)
	return result, nil
}
func (p *plannerRepository) CreateEvent(_ context.Context, id int, d domain.EventDraft) (domain.CalendarEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.writes++
	event := domain.CalendarEvent{ID: fmt.Sprint(p.writes), Title: d.Title, Start: d.Start, End: d.Start.Add(time.Duration(d.Duration * float64(time.Hour)))}
	p.events[id] = append(p.events[id], event)
	return event, nil
}

type plannerModel struct{}

func (plannerModel) Generate(context.Context, string) (string, error) {
	return `Предлагаю прогулку.|||CALENDAR_EVENT|||[{"is_event":true,"title":"Прогулка","start_time":"2026-11-24T18:00:00+03:00","duration":1}]`, nil
}
func TestIntegrationPlannerRequiresExplicitConfirmation(t *testing.T) {
	t.Parallel()
	repository := &plannerRepository{events: make(map[int][]domain.CalendarEvent)}
	calendar := calendar_service.New(repository)
	chat := chat_service.New(plannerModel{}, calendar)
	server := startApp(t, false, app.Options{Calendar: calendar, Chat: chat})
	alice, bob := newClient(t), newClient(t)
	login(t, alice, server.URL, "alice")
	login(t, bob, server.URL, "bob")
	call(t, newClient(t), "POST", server.URL+"/api/v1/ai/chat", `{"text":"План"}`, 401)
	response := call(t, alice, "POST", server.URL+"/api/v1/ai/chat", `{"text":"Прогулка завтра"}`, 200)
	var reply chat_service.Reply
	if err := json.NewDecoder(response.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.Drafts) != 1 {
		t.Fatalf("drafts=%v", reply.Drafts)
	}
	repository.mu.Lock()
	writes := repository.writes
	repository.mu.Unlock()
	if writes != 0 {
		t.Fatal("chat wrote events without confirmation")
	}
	draft := reply.Drafts[0]
	draft.RequestID = "test-confirmation"
	body, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	call(t, alice, "POST", server.URL+"/api/v1/calendar/events", string(body), 201)
	window := "?start=2026-11-23T00:00:00Z&end=2026-11-30T00:00:00Z"
	response = call(t, bob, "GET", server.URL+"/api/v1/calendar/events"+window, "", 200)
	var events []domain.CalendarEvent
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatal("Bob saw Alice's calendar")
	}
	response = call(t, alice, "GET", server.URL+"/api/v1/calendar/events"+window, "", 200)
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Title != "Прогулка" {
		t.Fatalf("events=%v", events)
	}
	call(t, alice, "GET", server.URL+"/api/v1/calendar/events?start=bad", "", 400)
}
func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func call(t *testing.T, client *http.Client, method, address, body string, status int) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, address, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	if resp.StatusCode != status {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: status %d, want %d; body %s", method, address, resp.StatusCode, status, data)
	}
	return resp
}
func login(t *testing.T, client *http.Client, base, code string) {
	t.Helper()
	loginAt(t, client, base, code, "/auth")
}

func loginAt(t *testing.T, client *http.Client, base, code, authPath string) {
	t.Helper()
	resp := call(t, client, "GET", base+authPath+"/google", "", 307)
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := location.Query().Get("state")
	if len(state) != 64 {
		t.Fatalf("OAuth state length = %d", len(state))
	}
	resp = call(t, client, "GET", base+authPath+"/callback?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(state), "", 303)
	found := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "jwt_token" {
			found = true
			if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 86400 {
				t.Errorf("unsafe session cookie: %+v", cookie)
			}
		}
	}
	if !found {
		t.Fatal("JWT cookie not issued")
	}
	if resp.Header.Get("Location") != "/" {
		t.Errorf("redirect = %q", resp.Header.Get("Location"))
	}
}

func TestIntegrationGoogleLoginAndProfile(t *testing.T) {
	t.Parallel()
	server := startApp(t, false)
	client := newClient(t)
	call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 401)
	call(t, client, "POST", server.URL+"/auth/demo", "", 404)
	call(t, client, "GET", server.URL+"/auth/callback?code=alice&state=forged", "", 400)
	login(t, client, server.URL, "alice")
	resp := call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 200)
	if resp.Header.Get("Content-Type") != "application/json" || resp.Header.Get("X-Request-ID") == "" {
		t.Errorf("missing response headers: %v", resp.Header)
	}
	var profile struct {
		Level, XP, Coins int
		XPForNextLevel   int `json:"xp_for_next_level"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatal(err)
	}
	if profile.Level != 1 || profile.XP != 0 || profile.Coins != 0 || profile.XPForNextLevel != 100 {
		t.Errorf("initial profile = %+v", profile)
	}
	call(t, client, "POST", server.URL+"/auth/logout", "", 204)
	call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 401)
	// A made-up JWT must never authenticate a caller.
	u, _ := url.Parse(server.URL)
	client.Jar.SetCookies(u, []*http.Cookie{{Name: "jwt_token", Value: "forged.jwt.value", Path: "/"}})
	call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 401)
	call(t, client, "GET", server.URL+"/", "", 200)
	call(t, client, "GET", server.URL+"/assets/app.js", "", 200)
}

func TestIntegrationRegisteredVersionedOAuthCallback(t *testing.T) {
	t.Parallel()
	server := startApp(t, false)
	client := newClient(t)
	loginAt(t, client, server.URL, "alice", "/api/v1/auth")
	call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 200)
}

func TestIntegrationStatsAccess(t *testing.T) {
	t.Parallel()
	server := startApp(t, false)
	alice, bob := newClient(t), newClient(t)
	login(t, alice, server.URL, "alice")
	login(t, bob, server.URL, "bob")
	base := server.URL + "/api/v1/gamification/stats"
	for _, body := range []string{`{`, `{}`, `{"title":"Сила","value":-1}`, `{"title":"Сила","description":""}`, `{"title":"Сила","user_id":999}`, `{"title":"Сила"} {}`, `{"title":" ","value":0}`} {
		call(t, alice, "POST", base, body, 400)
	}
	call(t, newClient(t), "POST", base, `{"title":"Сила"}`, 401)
	resp := call(t, alice, "POST", base, `{"title":"Сила","description":"Тренировки","value":3}`, 201)
	var created struct {
		ID                 int
		Title, Description string
		Value              int
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 || created.Title != "Сила" || created.Description != "Тренировки" || created.Value != 3 {
		t.Fatalf("created = %+v", created)
	}
	call(t, alice, "POST", base, `{"title":"Сила"}`, 409)
	resp = call(t, bob, "GET", base, "", 200)
	var bobStats []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&bobStats); err != nil {
		t.Fatal(err)
	}
	if bobStats == nil || len(bobStats) != 0 {
		t.Errorf("Bob sees Alice's stats: %s", bobStats)
	}
	call(t, bob, "DELETE", fmt.Sprintf("%s/%d", base, created.ID), "", 404)
	resp = call(t, alice, "GET", base, "", 200)
	var aliceStats []struct{ ID int }
	if err := json.NewDecoder(resp.Body).Decode(&aliceStats); err != nil {
		t.Fatal(err)
	}
	if len(aliceStats) != 1 || aliceStats[0].ID != created.ID {
		t.Fatalf("owner's stats = %v", aliceStats)
	}
	// A cross-site browser request cannot mutate an authenticated user's data.
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/%d", base, created.ID), nil)
	req.Header.Set("Origin", "https://other.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	csrfResp, err := alice.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	csrfResp.Body.Close()
	if csrfResp.StatusCode != 403 {
		t.Errorf("cross-site status = %d", csrfResp.StatusCode)
	}
	call(t, alice, "DELETE", fmt.Sprintf("%s/%d", base, created.ID), "", 204)
	call(t, alice, "DELETE", fmt.Sprintf("%s/%d", base, created.ID), "", 404)
	call(t, alice, "DELETE", base+"/invalid", "", 400)
}

func TestIntegrationOAuthStateAndDemo(t *testing.T) {
	t.Parallel()
	server := startApp(t, true)
	client := newClient(t)
	resp := call(t, client, "GET", server.URL+"/auth/google", "", 307)
	location, _ := url.Parse(resp.Header.Get("Location"))
	state := location.Query().Get("state")
	call(t, client, "GET", server.URL+"/auth/callback?code=alice&state=wrong", "", 400)
	// A failed callback consumes the state cookie; replay cannot sign in.
	call(t, client, "GET", server.URL+"/auth/callback?code=alice&state="+state, "", 400)
	call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 401)
	call(t, client, "POST", server.URL+"/auth/demo", "", 204)
	call(t, client, "GET", server.URL+"/api/v1/gamification/profile", "", 200)
}
