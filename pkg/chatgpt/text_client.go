package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type TextChatRepository interface {
	SaveSession(chatID int64, session domain.ChatSession)
	GetSession(chatID int64) (domain.ChatSession, bool)
}

type SettingsRepository interface {
	GetSetting(ctx context.Context, chatID int64, key string) (string, error)
}

type textClient struct {
	token        string
	hc           *http.Client
	chatRepo     TextChatRepository
	settingsRepo SettingsRepository
}

func NewTextClient(
	token string,
	chatRepo TextChatRepository,
	settingsRepo SettingsRepository,
) *textClient {
	return &textClient{
		token:        token,
		hc:           &http.Client{},
		chatRepo:     chatRepo,
		settingsRepo: settingsRepo,
	}
}

// GenerateSingleResponse function for HTTP API
func (c *textClient) GenerateSingleResponse(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *textClient) GenerateChatResponse(chatID int64, prompt string) (string, error) {
	// Get the session for the chat or create a new one.
	session, ok := c.chatRepo.GetSession(chatID)
	if !ok {
		systemPrompt, err := c.settingsRepo.GetSetting(context.TODO(), chatID, domain.SystemPromptKey)
		if err != nil {
			return "", fmt.Errorf("fetching system prompt: %v", err)
		}

		session = domain.ChatSession{
			ModelName: "gpt-4o",
			Messages: []domain.ChatMessage{
				{Role: "system", Content: systemPrompt},
			},
		}
	}

	// Prepare the request.
	chatMessage := domain.ChatMessage{Role: "user", Content: prompt}
	chatRequest := chatCompletionsRequest{
		Model:     session.ModelName,
		Messages:  append(session.Messages, chatMessage),
		MaxTokens: 1000,
	}

	// Send request to the API.
	url := "https://api.openai.com/v1/chat/completions"
	resp, err := c.sendRequest(url, chatRequest)
	if err != nil {
		return "", fmt.Errorf("sending request to %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Process the response.
	var chatResponse chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return "", fmt.Errorf("decoding response data: %v", err)
	}

	if len(chatResponse.Choices) > 0 && fmt.Sprint(chatResponse.Choices[0].Message.Content) != "" {
		// Update the session with new messages and save it.
		session.Messages = append(session.Messages, chatMessage, chatResponse.Choices[0].Message)
		c.chatRepo.SaveSession(chatID, session)

		return fmt.Sprint(chatResponse.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no completion response from API")
}

func (c *textClient) sendRequest(url string, chatRequest chatCompletionsRequest) (*http.Response, error) {
	body, err := json.Marshal(chatRequest)
	if err != nil {
		return nil, fmt.Errorf("marshaling chat request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
