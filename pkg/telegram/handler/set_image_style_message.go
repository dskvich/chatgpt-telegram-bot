package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setImageStyleMessage struct {
	client TelegramClient
}

func NewSetImageStyleMessage(
	client TelegramClient,
) *setImageStyleMessage {
	return &setImageStyleMessage{
		client: client,
	}
}

func (_ *setImageStyleMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/image_style")
}

func (s *setImageStyleMessage) Handle(u *tgbotapi.Update) {
	s.client.SendImageStyleMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
	})
}
