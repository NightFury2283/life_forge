package ai

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	log.Printf("Send message to AI, length: %d symbols", len(prompt))

	// if len(prompt) > 500 {
	// 	log.Printf("Start of the promt (1000 symbols): %s", prompt[:1000])
	// } else {
	// 	log.Printf("Full promt: %s", prompt)
	// }

	// TLS fix
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 60 * time.Second}

	// get token
	tokenForm := url.Values{}
	tokenForm.Set("scope", "GIGACHAT_API_PERS")

	tokenHttpReq, err := http.NewRequest("POST",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		strings.NewReader(tokenForm.Encode()))

	if err != nil {
		log.Printf("Error to create token request: %v", err)
		return "", fmt.Errorf("Token request create failed: %w", err)
	}

	tokenHttpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenHttpReq.Header.Set("RqUID", "123e4567-e89b-12d3-a456-426614174000")
	tokenHttpReq.Header.Set("Authorization", "Basic "+gg_cl.authKey)

	tokenResp, err := client.Do(tokenHttpReq)
	if err != nil {
		log.Printf("Error token request: %v", err)
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != 200 {
		body, _ := io.ReadAll(tokenResp.Body)
		log.Printf("Token Error %d: %s", tokenResp.StatusCode, string(body))
		return "", fmt.Errorf("Token http %d", tokenResp.StatusCode)
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		log.Printf("Token decode failed: %v", err)
		return "", fmt.Errorf("Token decode failed: %w", err)
	}

	log.Printf("Get token succesfully")

	//query to AI
	reqBody := GigaChatRequest{
		Model:    "GigaChat-Pro",
		Messages: []Message{{Role: "user", Content: prompt}},
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Error marshal json: %v", err)
		return "", fmt.Errorf("chat marshal failed: %w", err)
	}

	chatHttpReq, err := http.NewRequest("POST",
		"https://gigachat.devices.sberbank.ru/api/v1/chat/completions",
		bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("chat request create failed: %v", err)
		return "", fmt.Errorf("chat request create failed: %w", err)
	}

	chatHttpReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	chatHttpReq.Header.Set("Content-Type", "application/json")
	chatHttpReq.Header.Set("RqUID", "123e4567-e89b-12d3-a456-426614174001")

	log.Printf("Send query to chat...")
	resp, err := client.Do(chatHttpReq)
	if err != nil {
		log.Printf("chat request failed: %v", err)
		return "", fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("get AI answer, status: %d", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error to read response: %v", err)
		return "", fmt.Errorf("read response failed: %w", err)
	}

	var chatResp GigaChatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		log.Printf("Error to decode answer: %v", err)
		return "", fmt.Errorf("chat decode failed: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		log.Printf("no choices in response")
		return "", fmt.Errorf("no choices in response")
	}

	responseText := chatResp.Choices[0].Message.Content

	return responseText, nil
}
