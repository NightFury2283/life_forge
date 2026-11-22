package auth_repository_postgres

import (
	"bytes"
	"context"
	"errors"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	pool "github.com/NightFury2283/life_forge/internal/core/repository/postgres/pool"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/oauth2"
	"testing"
	"time"
)

type memoryPool struct {
	pool.Pool
	data map[int][]byte
}

func (p *memoryPool) OperationTimeout() time.Duration { return time.Second }
func (p *memoryPool) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	p.data[args[0].(int)] = bytes.Clone(args[1].([]byte))
	return pgconn.NewCommandTag("INSERT 1"), nil
}
func (p *memoryPool) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	return tokenRow{p.data[args[0].(int)]}
}

type tokenRow struct{ data []byte }

func (r tokenRow) Scan(dest ...any) error {
	if r.data == nil {
		return pgx.ErrNoRows
	}
	*dest[0].(*[]byte) = bytes.Clone(r.data)
	return nil
}
func TestEncryptedGoogleTokens(t *testing.T) {
	t.Parallel()
	p := &memoryPool{data: make(map[int][]byte)}
	store, err := NewTokenStore(p, "test-secret-of-at-least-thirty-two-bytes")
	if err != nil {
		t.Fatal(err)
	}
	input := &oauth2.Token{AccessToken: "access-secret", RefreshToken: "refresh-secret", Expiry: time.Now().Add(time.Hour)}
	if err := store.SaveGoogleToken(t.Context(), 7, input); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(p.data[7], []byte("secret")) {
		t.Fatal("token stored in plaintext")
	}
	got, err := store.GetGoogleToken(t.Context(), 7)
	if err != nil || got.AccessToken != input.AccessToken || got.RefreshToken != input.RefreshToken || !got.Expiry.Equal(input.Expiry) {
		t.Fatalf("token roundtrip failed: %v", err)
	}
	if err := store.SaveGoogleToken(t.Context(), 7, &oauth2.Token{AccessToken: "new-access"}); err != nil {
		t.Fatal(err)
	}
	got, err = store.GetGoogleToken(t.Context(), 7)
	if err != nil || got.RefreshToken != "refresh-secret" {
		t.Fatal("refresh token lost")
	}
	p.data[8] = bytes.Clone(p.data[7])
	if _, err := store.GetGoogleToken(t.Context(), 8); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatal("ciphertext accepted for a different user")
	}
	p.data[7][len(p.data[7])-1] ^= 1
	if _, err := store.GetGoogleToken(t.Context(), 7); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatal("tampered ciphertext accepted")
	}
	if _, err := store.GetGoogleToken(t.Context(), 9); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatal("missing account accepted")
	}
}
