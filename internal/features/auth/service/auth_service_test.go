package auth_service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	auth "github.com/NightFury2283/life_forge/internal/features/auth/service"
	"golang.org/x/oauth2"
)

type googleStub struct {
	url      func(string) string
	exchange func(context.Context, string) (*oauth2.Token, error)
	info     func(context.Context, *oauth2.Token) (*domain.GoogleUserInfo, error)
}

func (g googleStub) GetAuthURL(state string) string { return g.url(state) }
func (g googleStub) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.exchange(ctx, code)
}
func (g googleStub) GetGoogleUserInfo(ctx context.Context, tok *oauth2.Token) (*domain.GoogleUserInfo, error) {
	return g.info(ctx, tok)
}

type userRepoFunc func(context.Context, string, string, string) (domain.User, error)

func (f userRepoFunc) GetOrCreateUserByGoogleID(ctx context.Context, id, email, name string) (domain.User, error) {
	return f(ctx, id, email, name)
}

type tokenFunc func(int) (string, error)

func (f tokenFunc) GenerateToken(id int) (string, error) { return f(id) }

func TestLoginViaGoogle(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		failAt int
		calls  []string
	}{
		{"success", 0, []string{"exchange", "info", "repository", "sign"}},
		{"exchange_error", 1, []string{"exchange"}},
		{"user_info_error", 2, []string{"exchange", "info"}},
		{"repository_error", 3, []string{"exchange", "info", "repository"}},
		{"signing_error", 4, []string{"exchange", "info", "repository", "sign"}},
		{"missing_code", 5, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			cause := errors.New("dependency failed")
			var calls []string
			tok := &oauth2.Token{AccessToken: "google-token"}
			provider := googleStub{
				exchange: func(got context.Context, code string) (*oauth2.Token, error) {
					calls = append(calls, "exchange")
					if got != ctx || code != "code" {
						t.Fatalf("unexpected exchange arguments")
					}
					if tt.failAt == 1 {
						return nil, cause
					}
					return tok, nil
				},
				info: func(got context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
					calls = append(calls, "info")
					if got != ctx || token != tok {
						t.Fatal("unexpected user info arguments")
					}
					if tt.failAt == 2 {
						return nil, cause
					}
					return &domain.GoogleUserInfo{ID: "google-42", Email: "test@example.com", Name: "Test"}, nil
				},
			}
			repo := userRepoFunc(func(got context.Context, id, email, name string) (domain.User, error) {
				calls = append(calls, "repository")
				if got != ctx || id != "google-42" || email != "test@example.com" || name != "Test" {
					t.Fatal("unexpected repository arguments")
				}
				if tt.failAt == 3 {
					return domain.User{}, cause
				}
				return domain.User{ID: 42}, nil
			})
			signer := tokenFunc(func(id int) (string, error) {
				calls = append(calls, "sign")
				if id != 42 {
					t.Fatalf("user ID = %d", id)
				}
				if tt.failAt == 4 {
					return "", cause
				}
				return "jwt-token", nil
			})
			code := "code"
			if tt.failAt == 5 {
				code = ""
			}
			result, err := auth.NewAuthService(provider, repo, signer).LoginViaGoogle(ctx, code)
			if tt.failAt == 0 {
				if err != nil || result != "jwt-token" {
					t.Fatalf("LoginViaGoogle() = (%q, %v)", result, err)
				}
			} else {
				if result != "" || err == nil {
					t.Fatalf("LoginViaGoogle() = (%q, %v), want failure", result, err)
				}
				if tt.failAt != 5 && !errors.Is(err, cause) {
					t.Errorf("original error lost: %v", err)
				}
			}
			if !reflect.DeepEqual(calls, tt.calls) {
				t.Errorf("calls = %v, want %v", calls, tt.calls)
			}
		})
	}
}

func TestGetGoogleAuthURLForwardsState(t *testing.T) {
	t.Parallel()
	provider := googleStub{url: func(state string) string {
		if state != "random-state" {
			t.Fatalf("state = %q", state)
		}
		return "https://accounts.example/authorize"
	}}
	if got := auth.NewAuthService(provider, nil, nil).GetGoogleAuthURL("random-state"); got != "https://accounts.example/authorize" {
		t.Errorf("URL = %q", got)
	}
}
