package auth_repository_google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type GoogleAuthProvider struct {
	config *oauth2.Config
}

func NewGoogleAuthProvider(credentialsPath, redirectURL string) (*GoogleAuthProvider, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials.json: %w", err)
	}

	config, err := google.ConfigFromJSON(data,
		"openid",
		calendar.CalendarScope,
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	)
	if err != nil {
		return nil, fmt.Errorf("parse oauth config: %w", err)
	}

	// Keep the registered URI from credentials.json unless explicitly overridden.
	if redirectURL != "" {
		config.RedirectURL = redirectURL
	}
	return &GoogleAuthProvider{config: config}, nil
}

func (g *GoogleAuthProvider) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

func (g *GoogleAuthProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("Exchanging code for token: %w", err)
	}

	return token, nil
}

func (g *GoogleAuthProvider) GetGoogleUserInfo(
	ctx context.Context,
	token *oauth2.Token,
) (*domain.GoogleUserInfo, error) {

	client := g.config.Client(ctx, token)
	client.Timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Preserve the endpoint used by the original application. The alternate
	// Google host is tried only on a transport failure, never on rejected auth
	// or an invalid identity response.
	info, err := getGoogleUserInfo(ctx, client, "https://www.googleapis.com/oauth2/v2/userinfo")
	if err == nil {
		return info, nil
	}
	var transportErr *url.Error
	if ctx.Err() != nil || !errors.As(err, &transportErr) {
		return nil, err
	}
	fallback, fallbackErr := getGoogleUserInfo(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo")
	if fallbackErr != nil {
		return nil, errors.Join(err, fallbackErr)
	}
	return fallback, nil
}

func getGoogleUserInfo(ctx context.Context, client *http.Client, endpoint string) (*domain.GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create user info request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user info: unexpected HTTP status %d", resp.StatusCode)
	}

	var userInfo struct {
		ID      string `json:"id"`
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	id := userInfo.ID
	if endpoint == "https://openidconnect.googleapis.com/v1/userinfo" {
		id = userInfo.Subject
	}
	if id == "" {
		return nil, fmt.Errorf("Google user info is missing user ID")
	}
	return &domain.GoogleUserInfo{ID: id, Email: userInfo.Email, Name: userInfo.Name}, nil
}
