package chatgpt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	body := fmt.Sprintf(`{
		"model": "gpt-4-vision-preview",
		"messages": [
		  {
			"role": "user",
			"content": [
			  {
				"type": "text",
				"text": "%s"
			  },
			  {
				"type": "image_url",
				"image_url": {
				  "url": "data:image/jpeg;base64,%s"
				}
			  }
			]
		  }
		],
		"max_tokens": 300
	  }`, caption, base64image)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching usage data: %v", err)
	}

	var response chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decoding response data: %v", err)
	}

	return response.Choices[0].Message.Content, nil
}

type chatCompletionResponse struct {
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
