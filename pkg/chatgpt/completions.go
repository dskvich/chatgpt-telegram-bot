package chatgpt

import "github.com/dskvich/chatgpt-telegram-bot/pkg/domain"

type chatCompletionsRequest struct {
	Model     string               `json:"model"`
	Messages  []domain.ChatMessage `json:"messages"`
	MaxTokens int                  `json:"max_tokens"`
}

type userContent struct {
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
	Choices []struct {
		Message       domain.ChatMessage `json:"message"`
		FinishDetails struct {
			Type string `json:"type"`
			Stop string `json:"stop"`
		} `json:"finish_details"`
		Index int `json:"index"`
	} `json:"choices"`
}
