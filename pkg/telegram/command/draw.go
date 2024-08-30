package command

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

var drawSubstrings = []string{"рисуй", "draw"}

type DalleProvider interface {
	GenerateImage(prompt string) ([]byte, error)
}

type TextPromptSaver interface {
	Save(ctx context.Context, p *domain.Prompt) error
}

type draw struct {
	provider DalleProvider
	saver    TextPromptSaver
	client   TelegramClient
}

func NewDraw(
	provider DalleProvider,
	saver TextPromptSaver,
	client TelegramClient,
) *draw {
	return &draw{
		provider: provider,
		saver:    saver,
		client:   client,
	}
}

func (d *draw) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	lowerText := strings.ToLower(update.Message.Text)

	if strings.HasPrefix(lowerText, "/image") {
		return true
	}

	for _, substring := range drawSubstrings {
		if strings.Contains(lowerText, substring) {
			return true
		}
	}
	return false
}

func (d *draw) Execute(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID
	prompt := d.extractAfterSubstrings(update.Message.Text, drawSubstrings)

	if err := d.saver.Save(context.Background(), &domain.Prompt{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      prompt,
		FromUser:  fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
	}); err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to save prompt: %v", err),
		})
	}

	imgBytes, err := d.provider.GenerateImage(prompt)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		})
		return
	}

	d.client.SendImageMessage(domain.ImageMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Bytes:            imgBytes,
	})
}

func (d *draw) extractAfterSubstrings(s string, substrings []string) string {
	for _, substring := range substrings {
		index := strings.Index(strings.ToLower(s), strings.ToLower(substring))
		if index != -1 {
			return strings.TrimSpace(s[index+len(substring):])
		}
	}
	return s
}
