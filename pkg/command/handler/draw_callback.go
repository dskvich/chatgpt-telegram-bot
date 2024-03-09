package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
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
	outCh    chan<- domain.Message
}

func NewDrawCallback(
	provider DalleCallbackProvider,
	fetcher PromptFetcher,
	outCh chan<- domain.Message,
) *drawCallback {
	return &drawCallback{
		provider: provider,
		fetcher:  fetcher,
		outCh:    outCh,
	}
}

func (d *drawCallback) CanHandle(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, domain.DrawCallback)
}

func (d *drawCallback) Handle(update *tgbotapi.Update) {
	chatID := update.CallbackQuery.Message.Chat.ID
	messageID := update.CallbackQuery.Message.ReplyToMessage.MessageID

	prompt, err := d.fetcher.FetchPrompt(context.Background(), chatID, messageID)
	if err != nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to fetch prompt for message: %v", err),
		}
		return
	}

	if prompt == nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          "Упс! Я не могу найти исходный запрос для генерации похожего изображения. Пожалуйста, повторите ваш запрос.",
		}
		return
	}

	imgBytes, err := d.provider.GenerateImage(prompt.Text)
	if err != nil {
		d.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to generate image using Dall-E: %v", err),
		}
		return
	}

	d.outCh <- &domain.CallbackMessage{
		ID: update.CallbackQuery.ID,
	}

	d.outCh <- &domain.ImageMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Content:          imgBytes,
	}
}
