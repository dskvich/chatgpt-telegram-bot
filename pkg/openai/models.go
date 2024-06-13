package openai

import (
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

const (
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
	ChatMessageRoleTool      = "tool"
)

type chatCompletionsRequest struct {
	Model     string               `json:"model"`
	Messages  []domain.ChatMessage `json:"messages"`
	MaxTokens int                  `json:"max_tokens,omitempty"`
	Tools     []Tool               `json:"tools,omitempty"`
}

type Tool struct {
	Type     ToolType  `json:"type"`
	Function *Function `json:"function,omitempty"`
}

type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

type Function struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Parameters  jsonschema.Definition `json:"parameters"`
}

type chatCompletionsResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

type ChatCompletionChoice struct {
	Index        int                `json:"index"`
	Message      domain.ChatMessage `json:"message"`
	FinishReason string             `json:"finish_reason"`
}
