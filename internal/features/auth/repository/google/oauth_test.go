package auth_repository_google_test

import (
	"context"
	"errors"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	google "github.com/NightFury2283/life_forge/internal/features/auth/repository/google"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleUserInfo(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		status     int
		wantErr    bool
	}{
		{"valid_original_id", `{"id":"google-42","email":"test@example.com","name":"Test"}`, 200, false},
		{"missing_subject", `{"email":"test@example.com"}`, 200, true},
		{"invalid_json", `{`, 200, true},
		{"rejected_token", `{}`, 401, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "credentials.json")
			data := `{"web":{"client_id":"test","client_secret":"test","redirect_uris":["http://localhost/auth/callback"]}}`
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			provider, err := google.NewGoogleAuthProvider(path, "")
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://www.googleapis.com/oauth2/v2/userinfo" {
					t.Errorf("unexpected URL: %s", r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing access token")
				}
				if err := r.Context().Err(); err != nil {
					return nil, err
				}
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body)), Header: make(http.Header)}, nil
			})}
			ctx := context.WithValue(t.Context(), oauth2.HTTPClient, client)
			info, err := provider.GetGoogleUserInfo(ctx, &oauth2.Token{AccessToken: "test-token"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, want error %v", err, tt.wantErr)
			}
			if !tt.wantErr && (info.ID != "google-42" || info.Email != "test@example.com" || info.Name != "Test") {
				t.Errorf("user info = %+v", info)
			}
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel()
			_, err = provider.GetGoogleUserInfo(cancelCtx, &oauth2.Token{AccessToken: "test-token"})
			if !errors.Is(err, context.Canceled) {
				t.Errorf("cancelled request error = %v", err)
			}
		})
	}
}

func TestGoogleAuthRedirectURL(t *testing.T) {
	t.Parallel()
	const registered = "http://localhost:5050/api/v1/auth/callback"
	for _, tt := range []struct{ name, override, want string }{
		{"registered_credentials_uri", "", registered},
		{"explicit_override", "http://localhost:5050/auth/callback", "http://localhost:5050/auth/callback"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "credentials.json")
			data := `{"web":{"client_id":"test-client","client_secret":"test-secret","auth_uri":"https://accounts.example/authorize","token_uri":"https://accounts.example/token","redirect_uris":["` + registered + `"]}}`
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			provider, err := google.NewGoogleAuthProvider(path, tt.override)
			if err != nil {
				t.Fatal(err)
			}
			authURL, err := url.Parse(provider.GetAuthURL("test-state"))
			if err != nil {
				t.Fatal(err)
			}
			if got := authURL.Query().Get("redirect_uri"); got != tt.want {
				t.Errorf("redirect_uri = %q, want %q", got, tt.want)
			}
			if got := authURL.Query().Get("state"); got != "test-state" {
				t.Errorf("state = %q", got)
			}
		})
	}
}

func TestGoogleUserInfoNetworkFallback(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name          string
		fallbackFails bool
	}{{"alternate_host_succeeds", false}, {"both_hosts_fail", true}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "credentials.json")
			data := `{"web":{"client_id":"test","client_secret":"test","redirect_uris":["http://localhost/auth/callback"]}}`
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			provider, err := google.NewGoogleAuthProvider(path, "")
			if err != nil {
				t.Fatal(err)
			}
			var hosts []string
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				hosts = append(hosts, r.URL.Host)
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing access token")
				}
				if r.URL.Host == "www.googleapis.com" || tt.fallbackFails {
					return nil, io.EOF
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"sub":"google-42","name":"Test"}`)), Header: make(http.Header)}, nil
			})}
			ctx := context.WithValue(t.Context(), oauth2.HTTPClient, client)
			info, err := provider.GetGoogleUserInfo(ctx, &oauth2.Token{AccessToken: "test-token"})
			if tt.fallbackFails {
				if !errors.Is(err, io.EOF) || info != nil {
					t.Fatalf("result = (%v, %v)", info, err)
				}
			} else if err != nil || info == nil || info.ID != "google-42" {
				t.Fatalf("result = (%v, %v)", info, err)
			}
			if len(hosts) != 2 || hosts[0] != "www.googleapis.com" || hosts[1] != "openidconnect.googleapis.com" {
				t.Errorf("request hosts = %v", hosts)
			}
		})
	}
}
