package command

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ImageGenerator interface {
	GenerateImage(prompt string) ([]byte, error)
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

func (d *drawImage) IsCommand(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}
	return strings.HasPrefix(u.Message.Text, "/image") ||
		domain.CommandText(u.Message.Text).ContainsAny(domain.DrawKeywords)
}

func (d *drawImage) HandleCommand(u *tgbotapi.Update) {
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
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to save prompt: %v", err),
		})
	}

	d.generateAndSendImage(chatID, messageID, prompt)
}

func (d *drawImage) generateAndSendImage(chatID int64, messageID int, prompt string) {
	imgBytes, err := d.generator.GenerateImage(prompt)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to generate image: %v", err),
		})
		return
	}

	d.client.SendImageMessage(domain.ImageMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Bytes:            imgBytes,
	})
}

func (d *drawImage) IsCallback(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.RedrawCallback)
}

func (d *drawImage) HandleCallback(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	messageID := u.CallbackQuery.Message.ReplyToMessage.MessageID

	prompt, err := d.storage.FetchPrompt(context.Background(), chatID, messageID)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to fetch prompt: %v", err),
		})
		return
	}

	if prompt == nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             "Sorry, I can't find the original request for generating a similar image. Please try again.",
		})
		return
	}

	d.generateAndSendImage(chatID, messageID, prompt.Text)

	d.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: u.CallbackQuery.ID,
	})
}
