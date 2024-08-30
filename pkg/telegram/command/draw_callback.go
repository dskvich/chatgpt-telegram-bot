package command

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type DalleCallbackProvider interface {
	GenerateImage(prompt string) ([]byte, error)
}

type PromptFetcher interface {
	FetchPrompt(ctx context.Context, chatID int64, messageID int) (*domain.Prompt, error)
}

type drawCallback struct {
	provider DalleCallbackProvider
	fetcher  PromptFetcher
	client   TelegramClient
}

func NewDrawCallback(
	provider DalleCallbackProvider,
	fetcher PromptFetcher,
	client TelegramClient,
) *drawCallback {
	return &drawCallback{
		provider: provider,
		fetcher:  fetcher,
		client:   client,
	}
}

func (d *drawCallback) CanExecute(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, domain.RedrawCallback)
}

func (d *drawCallback) Execute(update *tgbotapi.Update) {
	chatID := update.CallbackQuery.Message.Chat.ID
	messageID := update.CallbackQuery.Message.ReplyToMessage.MessageID

	prompt, err := d.fetcher.FetchPrompt(context.Background(), chatID, messageID)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to fetch prompt for message: %v", err),
		})
		return
	}

	if prompt == nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             "Упс! Я не могу найти исходный запрос для генерации похожего изображения. Пожалуйста, повторите ваш запрос.",
		})
		return
	}

	imgBytes, err := d.provider.GenerateImage(prompt.Text)
	if err != nil {
		d.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		})
		return
	}

	d.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	d.client.SendImageMessage(domain.ImageMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Bytes:            imgBytes,
	})
}
