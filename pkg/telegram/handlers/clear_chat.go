package handlers

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ChatClearer interface {
	Clear(chatID int64, topicID int)
}

func ClearChat(clearer ChatClearer) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		slog.InfoContext(ctx, "Clearing chat")

		chatID := update.Message.Chat.ID
		topicID := update.Message.MessageThreadID

		clearer.Clear(chatID, topicID)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          chatID,
			MessageThreadID: topicID,
			Text:            "🧹 История очищена! Начните новый чат. 🚀",
		})
	}
}
