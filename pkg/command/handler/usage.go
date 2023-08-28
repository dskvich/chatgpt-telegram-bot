package handler

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type OpenAIUsageProvider interface {
	GetUsage() (string, error)
}

type usage struct {
	provider OpenAIUsageProvider
}

func NewUsage(provider OpenAIUsageProvider) *usage {
	return &usage{provider: provider}
}

func (u *usage) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && strings.HasPrefix(update.Message.Text, "/usage")
}

func (u *usage) Handle(update *tgbotapi.Update) domain.Message {
	response, err := u.provider.GetUsage()
	if err != nil {
		response = fmt.Sprintf("Failed to fetch OpenAI API usage info: %v", err)
	}
	return &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}
