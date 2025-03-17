package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/lo"
)

type SetTTLSettingsProvider interface {
	Get(ctx context.Context, chatID int64, topicID int) (*domain.Settings, error)
	Save(ctx context.Context, settings domain.Settings) error
}

func SetTTL(provider SetTTLSettingsProvider, supportedTTLOptions []time.Duration) bot.HandlerFunc {
	parseTTL := func(ttlRaw string) (time.Duration, error) {
		if !strings.HasPrefix(ttlRaw, domain.SetTTLCallbackPrefix) {
			return 0, fmt.Errorf("invalid format, expected prefix '%s'", domain.SetTTLCallbackPrefix)
		}

		ttlStr := strings.TrimPrefix(ttlRaw, domain.SetTTLCallbackPrefix)

		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return 0, err
		}

		if lo.Contains(supportedTTLOptions, ttl) {
			return ttl, nil
		}

		return 0, errors.New("unsupported ttl option")
	}

	shortDuration := func(d time.Duration) string {
		s := d.String()
		s = lo.Ternary(strings.HasSuffix(s, "m0s"), s[:len(s)-2], s)
		s = lo.Ternary(strings.HasSuffix(s, "h0m"), s[:len(s)-2], s)
		return s
	}

	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID := update.CallbackQuery.Message.Message.Chat.ID
		topicID := update.CallbackQuery.Message.Message.MessageThreadID

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		ttl, err := parseTTL(update.CallbackQuery.Data)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          chatID,
				MessageThreadID: topicID,
				Text:            fmt.Sprintf("❌ Не удалось извлечь TTL: %s", err),
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
		settings.TTL = ttl

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
			Text:            "✅ Время жизни чата (TTL) установлено: " + shortDuration(ttl),
		})
	}
}
