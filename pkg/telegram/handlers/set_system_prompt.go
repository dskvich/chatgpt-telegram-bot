package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

type SetSystemPromptSettingsProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Settings, error)
	Save(ctx context.Context, settings domain.Settings) error
}

type SetSystemPromptChatClearer interface {
	Clear(chatID int64, topicID int)
}

func SetSystemPrompt(provider SetSystemPromptSettingsProvider, clearer SetSystemPromptChatClearer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID
		prompt := update.Message.Text

		settings, err := provider.Get(ctx, chatID, topicID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось получить настройки: %s", err),
			})
			return
		}

		settings, _ = lo.Coalesce(settings, &domain.Settings{ChatID: chatID, TopicID: topicID})
		settings.SystemPrompt = prompt

		if err := provider.Save(ctx, *settings); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось сохранить настройки: %s", err),
			})
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "✅ Системная инструкция установлена: " + prompt,
		})

		clearer.Clear(chatID, topicID)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "🧹 История очищена! Начните новый чат. 🚀",
		})
	}
}
