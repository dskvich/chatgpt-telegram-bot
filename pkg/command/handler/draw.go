package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
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
	if update.Message == nil {
		return false
	}

	lowerText := strings.ToLower(update.Message.Text)
	for _, substring := range drawSubstrings {
		if strings.Contains(lowerText, substring) {
			return update.Message != nil
		}
	}
	return false
}

func (d *draw) Handle(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID
	prompt := d.extractAfterSubstrings(update.Message.Text, drawSubstrings)

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

func (d *draw) extractAfterSubstrings(s string, substrings []string) string {
	for _, substring := range substrings {
		index := strings.Index(strings.ToLower(s), strings.ToLower(substring))
		if index != -1 {
			return strings.TrimSpace(s[index+len(substring):])
		}
	}
	return s
}
