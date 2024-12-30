package openai

import (
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

const (
	chatMessageRoleDeveloper = "developer"
	chatMessageRoleUser      = "user"
	chatMessageRoleAssistant = "assistant"
	chatMessageRoleTool      = "tool"

	toolTypeFunction toolType = "function"
)

type chatCompletionsRequest struct {
	Model     string               `json:"model"`
	Messages  []domain.ChatMessage `json:"messages"`
	MaxTokens int                  `json:"max_tokens,omitempty"`
	Tools     []tool               `json:"tools,omitempty"`
}

type tool struct {
	Type     toolType  `json:"type"`
	Function *function `json:"function,omitempty"`
}

type toolType string

type function struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Parameters  jsonschema.Definition `json:"parameters"`
}

type chatCompletionsResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	Index        int                `json:"index"`
	Message      domain.ChatMessage `json:"message"`
	FinishReason string             `json:"finish_reason"`
}
