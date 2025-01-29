package handler

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type setImageStyleMessage struct {
	telegramClient TelegramClient
}

func NewSetImageStyleMessage(
	telegramClient TelegramClient,
) *setImageStyleMessage {
	return &setImageStyleMessage{
		telegramClient: telegramClient,
	}
}

func (*setImageStyleMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/image_style")
}

func (s *setImageStyleMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	options := map[string]string{
		"Яркий":        "image_style_vivid",
		"Естественный": "image_style_natural",
	}
	title := "Выберите стиль генерации изображений:"
	s.telegramClient.SendKeyboard(ctx, u.Message.Chat.ID, options, title)
}
