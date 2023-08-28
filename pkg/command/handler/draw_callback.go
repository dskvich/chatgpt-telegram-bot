package handler

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type DalleCallbackProvider interface {
	GenerateImage(prompt string) ([]byte, error)
}

type drawCallback struct {
	provider DalleCallbackProvider
}

func NewDrawCallback(provider DalleCallbackProvider) *draw {
	return &draw{provider: provider}
}

func (d *drawCallback) CanHandle(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && update.CallbackQuery.Data == domain.DrawCallback
}

func (d *drawCallback) Handle(update tgbotapi.Update) domain.Message {
	imgBytes, err := d.provider.GenerateImage(update.CallbackQuery.Message.Text)
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
