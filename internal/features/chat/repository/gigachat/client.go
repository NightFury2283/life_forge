package gigachat_repository

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

//go:embed certs/russian_trusted_root_ca.pem
var rootCA []byte

type Config struct{ AuthKey, Model, APIURL string }
type Client struct {
	key, model, chatURL, tokenURL string
	http                          *http.Client
	mu                            sync.Mutex
	token                         string
	expires                       time.Time
}

func New(c Config) (*Client, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(rootCA) {
		return nil, fmt.Errorf("invalid GigaChat CA certificate")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	if c.Model == "" {
		c.Model = "GigaChat-Pro"
	}
	if c.APIURL == "" {
		c.APIURL = "https://gigachat.devices.sberbank.ru/api/v1"
	}
	u, err := url.Parse(c.APIURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, fmt.Errorf("GigaChat API URL must use HTTPS")
	}
	return &Client{key: strings.TrimPrefix(c.AuthKey, "Basic "), model: c.Model, chatURL: strings.TrimRight(c.APIURL, "/") + "/chat/completions", tokenURL: "https://ngw.devices.sberbank.ru:9443/api/v2/oauth", http: &http.Client{Transport: transport, Timeout: 60 * time.Second}}, nil
}
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Add(time.Minute).Before(c.expires) {
		return c.token, nil
	}
	if c.key == "" {
		return "", fmt.Errorf("GIGACHAT_AUTH_KEY is not configured")
	}
	form := url.Values{"scope": {"GIGACHAT_API_PERS"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+c.key)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("RqUID", uuid.NewString())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("GigaChat token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GigaChat token HTTP %d", resp.StatusCode)
	}
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", fmt.Errorf("empty GigaChat access token")
	}
	c.token = data.AccessToken
	c.expires = time.UnixMilli(data.ExpiresAt)
	if data.ExpiresAt == 0 {
		c.expires = time.Now().Add(29 * time.Minute)
	}
	return c.token, nil
}
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]any{"model": c.model, "messages": []map[string]string{{"role": "user", "content": prompt}}, "stream": false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("RqUID", uuid.NewString())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("GigaChat completion: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			c.mu.Lock()
			if c.token == token {
				c.token = ""
			}
			c.mu.Unlock()
		}
		return "", fmt.Errorf("GigaChat completion HTTP %d", resp.StatusCode)
	}
	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Choices) == 0 || data.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty GigaChat response")
	}
	return data.Choices[0].Message.Content, nil
}
