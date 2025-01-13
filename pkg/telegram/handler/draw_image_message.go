package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ImageGenerator interface {
	GenerateImage(chatID int64, prompt string) ([]byte, error)
}

type PromptStorage interface {
	SavePrompt(ctx context.Context, p *domain.Prompt) error
	FetchPrompt(ctx context.Context, chatID int64, messageID int) (*domain.Prompt, error)
}

type drawImage struct {
	generator ImageGenerator
	storage   PromptStorage
	client    TelegramClient
}

func NewDrawImage(
	generator ImageGenerator,
	storage PromptStorage,
	client TelegramClient,
) *drawImage {
	return &drawImage{
		generator: generator,
		storage:   storage,
		client:    client,
	}
}

func (_ *drawImage) CanHandle(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}
	return strings.HasPrefix(u.Message.Text, "/image") ||
		domain.CommandText(u.Message.Text).ContainsAny(domain.DrawKeywords)
}

func (d *drawImage) Handle(u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	messageID := u.Message.MessageID
	prompt := domain.CommandText(u.Message.Text).ExtractAfterKeywords(domain.DrawKeywords)

	if err := d.storage.SavePrompt(context.Background(), &domain.Prompt{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      prompt,
		FromUser:  fmt.Sprintf("%s %s", u.Message.From.FirstName, u.Message.From.LastName),
	}); err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to save prompt: %v", err),
		})
	}

	d.generateAndSendImage(chatID, prompt)
}

func (d *drawImage) generateAndSendImage(chatID int64, prompt string) {
	imgBytes, err := d.generator.GenerateImage(chatID, prompt)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to generate image: %v", err),
		})
		return
	}

	d.client.SendImageMessage(domain.ImageMessage{
		ChatID: chatID,
		Bytes:  imgBytes,
	})
}
