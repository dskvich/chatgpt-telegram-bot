package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type VisionChatRepository interface {
	SaveSession(chatID int64, session domain.ChatSession)
	GetSession(chatID int64) (domain.ChatSession, bool)
	RemoveSession(chatID int64)
}

type visionClient struct {
	token string
	hc    *http.Client
	repo  VisionChatRepository
}

func NewVisionClient(token string, repo VisionChatRepository) *visionClient {
	return &visionClient{
		token: token,
		hc:    &http.Client{},
		repo:  repo,
	}
}

func (c *visionClient) RecognizeImage(chatID int64, base64image, caption string) (string, error) {
	if caption == "" {
		caption = "Что на этом рисунке?"
	}

	// Start a new session when a new image is sent
	c.repo.RemoveSession(chatID)
	session := domain.ChatSession{
		ModelName: "gpt-4-turbo",
		Messages: []domain.ChatMessage{
			{
				Role: "user",
				Content: []userContent{
					{Type: "text", Text: caption},
					{Type: "image_url", ImageUrl: &imageUrl{Url: "data:image/jpeg;base64," + base64image}},
				},
			},
		},
	}

	// Prepare the request.
	chatRequest := chatCompletionsRequest{
		Model:     session.ModelName,
		Messages:  session.Messages,
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
		session.Messages = append(session.Messages, chatResponse.Choices[0].Message)
		c.repo.SaveSession(chatID, session)

		return fmt.Sprint(chatResponse.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no completion response from API")
}

func (c *visionClient) sendRequest(url string, chatRequest chatCompletionsRequest) (*http.Response, error) {
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
