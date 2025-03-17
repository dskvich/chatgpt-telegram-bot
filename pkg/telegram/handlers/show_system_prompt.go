package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ShowSystemPromptSettingsProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Settings, error)
}

func ShowSystemPrompt(provider ShowSystemPromptSettingsProvider) bot.HandlerFunc {
	const editButtonText = "Редактировать"

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		settings, err := provider.Get(ctx, chatID, topicID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          update.Message.Chat.ID,
				MessageThreadID: update.Message.MessageThreadID,
				Text:            fmt.Sprintf("❌ Не удалось получить настройки: %s", err),
			})
			return
		}

		prompt := "Отсутсвует"
		if settings != nil && settings.SystemPrompt != "" {
			prompt = settings.SystemPrompt
		}

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: editButtonText, CallbackData: domain.SetSystemPromptCallbackPrefix + editButtonText},
				},
			},
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            fmt.Sprintf("🧠 Текущая системная инструкция:\n%s", prompt),
			ReplyMarkup:     kb,
		})
	}
}
