package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type DalleProvider interface {
	GenerateImage(prompt string) ([]byte, error)
}

type TextPromptSaver interface {
	Save(ctx context.Context, p *domain.Prompt) error
}

type draw struct {
	provider DalleProvider
	saver    TextPromptSaver
	outCh    chan<- domain.Message
}

func NewDraw(
	provider DalleProvider,
	saver TextPromptSaver,
	outCh chan<- domain.Message,
) *draw {
	return &draw{
		provider: provider,
		saver:    saver,
		outCh:    outCh,
	}
}

func (d *draw) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && strings.Contains(strings.ToLower(update.Message.Text), "рисуй")
}

func (d *draw) Handle(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID
	prompt := update.Message.Text

	if err := d.saver.Save(context.Background(), &domain.Prompt{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      prompt,
		FromUser:  fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
	}); err != nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to save prompt: %v", err),
		}
	}

	imgBytes, err := d.provider.GenerateImage(prompt)
	if err != nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		}
		return
	}

	d.outCh <- &domain.ImageMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Content:          imgBytes,
	}
}
