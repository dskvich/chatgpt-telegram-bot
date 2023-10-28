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
	outCh    chan<- domain.Message
}

func NewDraw(
	provider DalleProvider,
	outCh chan<- domain.Message,
) *draw {
	return &draw{
		provider: provider,
		outCh:    outCh,
	}
}

func (d *draw) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && strings.Contains(strings.ToLower(update.Message.Text), "рисуй")
}

func (d *draw) Handle(update *tgbotapi.Update) {
	imgBytes, err := d.provider.GenerateImage(update.Message.Text)
	if err != nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           update.Message.Chat.ID,
			ReplyToMessageID: update.Message.MessageID,
			Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		}
		return
	}

	d.outCh <- &domain.ImageMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          imgBytes,
	}
}
