package core_auth_jwt_test

import (
	"strconv"
	"testing"
	"time"

	tokens "github.com/NightFury2283/life_forge/internal/core/auth/jwt"
	"github.com/golang-jwt/jwt/v5"
)

const secret = "test-secret-at-least-thirty-two-bytes-long"

func TestTokenServiceRoundTrip(t *testing.T) {
	t.Parallel()
	service := tokens.NewTokenService(tokens.JWTConfig{SecretKey: secret})
	before := time.Now()
	raw, err := service.GenerateToken(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := service.ParseToken(raw)
	if err != nil || id != 42 {
		t.Fatalf("ParseToken() = (%d, %v), want (42, nil)", id, err)
	}
	claims := &tokens.Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return []byte(secret), nil })
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Method != jwt.SigningMethodHS256 {
		t.Errorf("signing method = %s", parsed.Method.Alg())
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatal("missing expiry or issue time")
	}
	if claims.ExpiresAt.Time.Before(before.Add(24*time.Hour-time.Second)) || claims.ExpiresAt.Time.After(time.Now().Add(24*time.Hour)) {
		t.Errorf("unexpected expiry: %v", claims.ExpiresAt)
	}
	if claims.Subject != "42" || claims.Issuer != "life-forge" {
		t.Errorf("unexpected identity: %+v", claims)
	}
}

func TestParseTokenRejectsInvalidTokens(t *testing.T) {
	t.Parallel()
	service := tokens.NewTokenService(tokens.JWTConfig{SecretKey: secret})
	valid := func() tokens.Claims {
		return tokens.Claims{UserID: 7, RegisteredClaims: jwt.RegisteredClaims{Issuer: "life-forge", Subject: strconv.Itoa(7), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}}
	}
	tests := []struct {
		name   string
		method jwt.SigningMethod
		key    any
		change func(*tokens.Claims)
	}{
		{"wrong_secret", jwt.SigningMethodHS256, []byte("a-different-secret"), nil},
		{"wrong_algorithm", jwt.SigningMethodHS512, []byte(secret), nil},
		{"unsigned", jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, nil},
		{"expired", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) }},
		{"missing_expiry", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.ExpiresAt = nil }},
		{"future_issue_time", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Hour)) }},
		{"wrong_issuer", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.Issuer = "other-app" }},
		{"zero_user", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.UserID = 0; c.Subject = "0" }},
		{"negative_user", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.UserID = -1; c.Subject = "-1" }},
		{"mismatching_subject", jwt.SigningMethodHS256, []byte(secret), func(c *tokens.Claims) { c.Subject = "8" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims := valid()
			if tt.change != nil {
				tt.change(&claims)
			}
			raw, err := jwt.NewWithClaims(tt.method, &claims).SignedString(tt.key)
			if err != nil {
				t.Fatal(err)
			}
			id, err := service.ParseToken(raw)
			if err == nil || id != 0 {
				t.Fatalf("ParseToken() = (%d, %v), want rejected token", id, err)
			}
		})
	}
	for _, raw := range []string{"", "not.a.jwt", "invalid", "eyJhbGciOiJIUzI1NiJ9.e30.tampered"} {
		t.Run("malformed/"+raw, func(t *testing.T) {
			if id, err := service.ParseToken(raw); err == nil || id != 0 {
				t.Fatalf("ParseToken(%q) = (%d, %v)", raw, id, err)
			}
		})
	}
}

func TestGenerateTokenRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, key string
		id        int
	}{{"zero_user", secret, 0}, {"negative_user", secret, -1}, {"empty_secret", "", 1}, {"short_secret", "short", 1}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw, err := tokens.NewTokenService(tokens.JWTConfig{SecretKey: tt.key}).GenerateToken(tt.id)
			if err == nil || raw != "" {
				t.Fatalf("GenerateToken() = (%q, %v)", raw, err)
			}
		})
	}
}
