package ai

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GigaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type GigaChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type GigaChatClient struct {
	authKey string
}

func NewGigaChatClient(authKey string) *GigaChatClient {
	return &GigaChatClient{authKey: authKey}
}

func (gg_cl *GigaChatClient) Generate(prompt string) (string, error) {
	// TLS fix
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 60 * time.Second}

	tokenForm := url.Values{}
	tokenForm.Set("scope", "GIGACHAT_API_PERS")

	tokenHttpReq, err := http.NewRequest("POST",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		strings.NewReader(tokenForm.Encode()))

	if err != nil {
		return "", fmt.Errorf("Token request create failed: %w", err)
	}

	tokenHttpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenHttpReq.Header.Set("RqUID", "123e4567-e89b-12d3-a456-426614174000")
	tokenHttpReq.Header.Set("Authorization", "Basic "+gg_cl.authKey)

	tokenResp, err := client.Do(tokenHttpReq)

	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != 200 {
		body, _ := io.ReadAll(tokenResp.Body)
		fmt.Printf("Token error %d: %s\n", tokenResp.StatusCode, string(body))
		return "", fmt.Errorf("Token http %d", tokenResp.StatusCode)
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("Token decode failed: %w", err)
	}

	reqBody := GigaChatRequest{
		Model:    "GigaChat-Pro",
		Messages: []Message{{Role: "user", Content: prompt}},
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("chat marshal failed: %w", err)
	}

	chatHttpReq, err := http.NewRequest("POST",
		"https://gigachat.devices.sberbank.ru/api/v1/chat/completions",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("chat request create failed: %w", err)
	}
	chatHttpReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	chatHttpReq.Header.Set("Content-Type", "application/json")
	chatHttpReq.Header.Set("RqUID", "123e4567-e89b-12d3-a456-426614174001")

	resp, err := client.Do(chatHttpReq)
	if err != nil {
		return "", fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	var chatResp GigaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Chat decode error: %s\n", string(body))
		return "", fmt.Errorf("chat decode failed: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
