package gigachat_repository

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentGenerateCachesToken(t *testing.T) {
	t.Parallel()
	var tokens atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			tokens.Add(1)
			if r.Header.Get("Authorization") != "Basic key" || r.Header.Get("RqUID") == "" {
				t.Error("invalid token headers")
			}
			fmt.Fprintf(w, `{"access_token":"access","expires_at":%d}`, time.Now().Add(time.Hour).UnixMilli())
		case "/chat":
			if r.Header.Get("Authorization") != "Bearer access" {
				t.Error("missing bearer token")
			}
			fmt.Fprint(w, `{"choices":[{"message":{"content":"Привет"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := &Client{key: "key", model: "test", tokenURL: server.URL + "/token", chatURL: server.URL + "/chat", http: server.Client()}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			got, err := c.Generate(context.Background(), "Hello")
			if err != nil || got != "Привет" {
				t.Errorf("Generate() = %q, %v", got, err)
			}
		})
	}
	wg.Wait()
	if tokens.Load() != 1 {
		t.Errorf("token calls = %d, want 1", tokens.Load())
	}
}
func TestOfficialCAParses(t *testing.T) {
	t.Parallel()
	client, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	transport := client.http.Transport.(*http.Transport)
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS verification disabled")
	}
}
