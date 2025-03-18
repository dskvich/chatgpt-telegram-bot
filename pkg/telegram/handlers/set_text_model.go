package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

type SetTextModelSettingsProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Settings, error)
	Save(ctx context.Context, settings domain.Settings) error
}

type SetTextModelChatClearer interface {
	Clear(chatID int64, topicID int)
}

func SetTextModel(provider SetTextModelSettingsProvider, clearer SetTextModelChatClearer, supportedTextModels []string) bot.HandlerFunc {
	parseTextModel := func(modelRaw string) (string, error) {
		if !strings.HasPrefix(modelRaw, domain.SetTextModelCallbackPrefix) {
			return "", fmt.Errorf("invalid format, expected prefix '%s'", domain.SetTextModelCallbackPrefix)
		}

		model := strings.TrimPrefix(modelRaw, domain.SetTextModelCallbackPrefix)

		if lo.Contains(supportedTextModels, model) {
			return model, nil
		}

		return "", errors.New("unsupported model")
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.CallbackQuery.Message.Message.Chat.ID
		topicID := update.CallbackQuery.Message.Message.MessageThreadID

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		model, err := parseTextModel(update.CallbackQuery.Data)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось извлечь текстовую модель: %s", err),
			})
			return
		}

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
		settings.TextModel = model

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
			Text:            "✅ Модель установлена: " + model,
		})

		clearer.Clear(chatID, topicID)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "🧹 История очищена! Начните новый чат. 🚀",
		})
	}
}
