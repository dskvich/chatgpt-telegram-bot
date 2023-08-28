package handler

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type DalleProvider interface {
	GenerateImage(prompt string) ([]byte, error)
}

type draw struct {
	provider DalleProvider
}

func NewDraw(provider DalleProvider) *draw {
	return &draw{provider: provider}
}

func (d *draw) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && strings.Contains(strings.ToLower(update.Message.Text), "рисуй")
}

func (d *draw) Handle(update *tgbotapi.Update) domain.Message {
	imgBytes, err := d.provider.GenerateImage(update.Message.Text)
	if err != nil {
		return &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		}
	}
	return &domain.ImageMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          imgBytes,
	}
}
