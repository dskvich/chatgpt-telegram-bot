package handler

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type GptProvider interface {
	GenerateChatResponse(chatID int64, prompt string) (string, error)
}

type gpt struct {
	provider GptProvider
}

func NewGpt(provider GptProvider) *gpt {
	return &gpt{provider: provider}
}

func (_ *gpt) CanHandle(_ *tgbotapi.Update) bool {
	return true
}

func (g *gpt) Handle(update *tgbotapi.Update) domain.Message {
	response, err := g.provider.GenerateChatResponse(update.Message.Chat.ID, update.Message.Text)
	if err != nil {
		response = fmt.Sprintf("Failed to get response from ChatGPT: %v", err)
	}
	return &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}