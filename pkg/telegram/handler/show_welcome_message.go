package handler

import (
	"context"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type showWelcomeMessage struct {
	client TelegramClient
}

func NewShowWelcomeMessage(
	client TelegramClient,
) *showWelcomeMessage {
	return &showWelcomeMessage{
		client: client,
	}
}

func (*showWelcomeMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && (strings.HasPrefix(u.Message.Text, "/start") ||
		strings.Contains(strings.ToLower(u.Message.Text), "что ты умеешь") ||
		strings.Contains(strings.ToLower(u.Message.Text), "что ты можешь"))
}

func (s *showWelcomeMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	text := `👋 Я твой ChatGPT Telegram-бот. Вот что умею:

❓ Отвечаю на вопросы. Напиши "новый чат" для очистки истории.
🎨 Рисую картинки. Начни запрос с "нарисуй".
🎙 Понимаю голосовые сообщения.
📷 Распознаю картинки.`

	s.client.SendResponse(ctx, u.Message.Chat.ID, &domain.Response{Text: text})
}
