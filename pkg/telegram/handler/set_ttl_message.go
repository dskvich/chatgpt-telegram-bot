package handler

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type setTTLMessage struct {
	telegramClient TelegramClient
}

func NewSetTTLMessage(
	telegramClient TelegramClient,
) *setTTLMessage {
	return &setTTLMessage{
		telegramClient: telegramClient,
	}
}

func (*setTTLMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/ttl")
}

func (s *setTTLMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	options := map[string]string{
		"15 минут":  "ttl_15m",
		"1 час":     "ttl_1h",
		"8 часов":   "ttl_8h",
		"Отключено": "ttl_disabled",
	}
	title := "Выберите опцию TTL:"
	s.telegramClient.SendKeyboard(ctx, u.Message.Chat.ID, options, title)
}
