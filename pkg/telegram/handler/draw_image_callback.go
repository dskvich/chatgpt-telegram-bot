package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type CallbackImageGenerator interface {
	GenerateImage(chatID int64, prompt string) ([]byte, error)
}

type CallbackPromptStorage interface {
	SavePrompt(ctx context.Context, p *domain.Prompt) error
	FetchPrompt(ctx context.Context, chatID int64, messageID int) (*domain.Prompt, error)
}

type drawImageCallback struct {
	generator CallbackImageGenerator
	storage   CallbackPromptStorage
	client    TelegramClient
}

func NewDrawImageCallback(
	generator CallbackImageGenerator,
	storage CallbackPromptStorage,
	client TelegramClient,
) *drawImageCallback {
	return &drawImageCallback{
		generator: generator,
		storage:   storage,
		client:    client,
	}
}

func (_ *drawImageCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.RedrawCallback)
}

func (d *drawImageCallback) Handle(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	messageID := u.CallbackQuery.Message.ReplyToMessage.MessageID
	//TODO: panics after removing replyTo... need to get initial request from another place

	prompt, err := d.storage.FetchPrompt(context.Background(), chatID, messageID)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to fetch prompt: %v", err),
		})
		return
	}

	if prompt == nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   "Sorry, I can't find the original request for generating a similar image. Please try again.",
		})
		return
	}

	d.generateAndSendImage(chatID, prompt.Text)

	d.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: u.CallbackQuery.ID,
	})
}

func (d *drawImageCallback) generateAndSendImage(chatID int64, prompt string) {
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