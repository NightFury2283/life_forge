package auth_repository_postgres

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	pool "github.com/NightFury2283/life_forge/internal/core/repository/postgres/pool"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

type TokenStore struct {
	pool   pool.Pool
	cipher cipher.AEAD
}

func NewTokenStore(p pool.Pool, secret string) (*TokenStore, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("token encryption requires at least 32 secret bytes")
	}
	key := sha256.Sum256([]byte("life-forge/google-tokens/" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &TokenStore{pool: p, cipher: aead}, nil
}
func (s *TokenStore) SaveGoogleToken(ctx context.Context, userID int, token *oauth2.Token) error {
	ctx, cancel := context.WithTimeout(ctx, s.pool.OperationTimeout())
	defer cancel()
	if token == nil || token.AccessToken == "" {
		return core_errors.ErrInvalidArgument
	}
	copyToken := *token
	if copyToken.RefreshToken == "" {
		old, err := s.GetGoogleToken(ctx, userID)
		if err == nil {
			copyToken.RefreshToken = old.RefreshToken
		} else if !errors.Is(err, core_errors.ErrUnauthorized) {
			return err
		}
	}
	data, err := json.Marshal(copyToken)
	if err != nil {
		return err
	}
	nonce := make([]byte, s.cipher.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	sealed := s.cipher.Seal(nonce, nonce, data, []byte(strconv.Itoa(userID)))
	_, err = s.pool.Exec(ctx, `INSERT INTO lifeforge.google_tokens(user_id,encrypted_token) VALUES($1,$2)
	 ON CONFLICT(user_id) DO UPDATE SET encrypted_token=EXCLUDED.encrypted_token,updated_at=CURRENT_TIMESTAMP`, userID, sealed)
	if err != nil {
		return fmt.Errorf("store Google token: %w", err)
	}
	return nil
}
func (s *TokenStore) GetGoogleToken(ctx context.Context, userID int) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, s.pool.OperationTimeout())
	defer cancel()
	var data []byte
	if err := s.pool.QueryRow(ctx, `SELECT encrypted_token FROM lifeforge.google_tokens WHERE user_id=$1`, userID).Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reconnect Google to enable calendar: %w", core_errors.ErrUnauthorized)
		}
		return nil, err
	}
	n := s.cipher.NonceSize()
	if len(data) < n {
		return nil, fmt.Errorf("invalid encrypted Google credentials")
	}
	plain, err := s.cipher.Open(nil, data[:n], data[n:], []byte(strconv.Itoa(userID)))
	if err != nil {
		return nil, fmt.Errorf("decrypt Google credentials; reconnect Google: %w", core_errors.ErrUnauthorized)
	}
	var token oauth2.Token
	if err = json.Unmarshal(plain, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
