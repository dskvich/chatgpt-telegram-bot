package tools

import (
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveSession(chatID int64)
}

type clearChatSession struct {
	remover MessagesRemover
}

func NewClearChatSession(remover MessagesRemover) *clearChatSession {
	return &clearChatSession{
		remover: remover,
	}
}

func (c *clearChatSession) Name() string {
	return "clear_chat_session"
}

func (c *clearChatSession) Description() string {
	return "Removes all chat session history to start a new conversation from scratch"
}

func (c *clearChatSession) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
	}
}

func (c *clearChatSession) Function(chatID int64) (string, error) {
	c.remover.RemoveSession(chatID)

	return "", domain.ErrSessionInvalidated
}
