package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type showInfoMessage struct {
	client TelegramClient
}

func NewShowInfoMessage(
	client TelegramClient,
) *showInfoMessage {
	return &showInfoMessage{
		client: client,
	}
}

func (*showInfoMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && (strings.HasPrefix(u.Message.Text, "/start") ||
		strings.Contains(strings.ToLower(u.Message.Text), "что ты умеешь") ||
		strings.Contains(strings.ToLower(u.Message.Text), "что ты можешь"))
}

func (s *showInfoMessage) Handle(u *tgbotapi.Update) {
	s.client.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   domain.WelcomeMessage,
	})
}
