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
	outCh    chan<- domain.Message
}

func NewDrawCallback(
	provider DalleCallbackProvider,
	outCh chan<- domain.Message,
) *drawCallback {
	return &drawCallback{
		provider: provider,
		outCh:    outCh,
	}
}

func (d *drawCallback) CanHandle(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && update.CallbackQuery.Data == domain.DrawCallback
}

func (d *drawCallback) Handle(update *tgbotapi.Update) {
	imgBytes, err := d.provider.GenerateImage(update.CallbackQuery.Message.ReplyToMessage.Text)
	if err != nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           update.CallbackQuery.Message.Chat.ID,
			ReplyToMessageID: update.CallbackQuery.Message.ReplyToMessage.MessageID,
			Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		}
		return
	}

	d.outCh <- &domain.CallbackMessage{
		ID: update.CallbackQuery.ID,
	}

	d.outCh <- &domain.ImageMessage{
		ChatID:           update.CallbackQuery.Message.Chat.ID,
		ReplyToMessageID: update.CallbackQuery.Message.ReplyToMessage.MessageID,
		Content:          imgBytes,
	}
}
