package auth_repository_google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type GoogleAuthProvider struct {
	config *oauth2.Config
}

func NewGoogleAuthProvider(credentialsPath string) (*GoogleAuthProvider, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials.json: %w", err)
	}

	config, err := google.ConfigFromJSON(data,
		calendar.CalendarScope,
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	)
	if err != nil {
		return nil, fmt.Errorf("parse oauth config: %w", err)
	}

	return &GoogleAuthProvider{config: config}, nil
}

func (g *GoogleAuthProvider) GetAuthURL() string {
	// Переопределяем config с правильными scope
	g.config.Scopes = []string{
		calendar.CalendarScope,
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

	return g.config.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

func (g *GoogleAuthProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	g.config.Scopes = []string{
		calendar.CalendarScope,
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

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

	// Пробуем получить userinfo
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user info, got unexpected status: %w", err)
	}

	var userInfo domain.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	return &userInfo, nil
}
