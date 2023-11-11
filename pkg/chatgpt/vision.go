package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type visionClient struct {
	token string
	hc    *http.Client
}

func NewVisionClient(token string) *visionClient {
	return &visionClient{
		token: token,
		hc:    &http.Client{},
	}
}

func (c *visionClient) RecognizeImage(chatID int64, base64image, caption string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	req, err := c.prepareRequest(base64image, caption)
	if err != nil {
		return "", fmt.Errorf("preparing request: %v", err)
	}

	resp, err := c.sendRequest(url, req)
	if err != nil {
		return "", fmt.Errorf("sending request to %s: %v", url, err)
	}
	defer resp.Body.Close()

	return c.processResponse(resp)
}

func (c *visionClient) prepareRequest(base64image, text string) ([]byte, error) {
	url := "data:image/jpeg;base64," + base64image

	chatRequest := chatCompletionsRequest{
		Model: "gpt-4-vision-preview",
		Messages: []chatCompletionMessage{
			{
				Role: "user",
				Content: []messageContent{
					{Type: "text", Text: text},
					{Type: "image_url", ImageUrl: &imageUrl{Url: url}},
				},
			},
		},
		MaxTokens: 300,
	}

	body, err := json.Marshal(chatRequest)
	if err != nil {
		return nil, fmt.Errorf("marshaling chat request: %v", err)
	}

	return body, nil
}

func (c *visionClient) sendRequest(url string, body []byte) (*http.Response, error) {
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

func (c *visionClient) processResponse(resp *http.Response) (string, error) {
	var response chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decoding response data: %v", err)
	}

	if len(response.Choices) > 0 && response.Choices[0].Message.Content != "" {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no completion response from API")
}

type chatCompletionsRequest struct {
	Model     string                  `json:"model"`
	Messages  []chatCompletionMessage `json:"messages"`
	MaxTokens int                     `json:"max_tokens"`
}

type chatCompletionMessage struct {
	Role    string           `json:"role"`
	Content []messageContent `json:"content"`
}

type messageContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageUrl *imageUrl `json:"image_url,omitempty"`
}

type imageUrl struct {
	Url string `json:"url"`
}

type chatCompletionsResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishDetails struct {
			Type string `json:"type"`
			Stop string `json:"stop"`
		} `json:"finish_details"`
		Index int `json:"index"`
	} `json:"choices"`
}
