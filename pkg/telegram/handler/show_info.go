package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type showInfo struct {
	client TelegramClient
}

func NewShowInfo(
	client TelegramClient,
) *showInfo {
	return &showInfo{
		client: client,
	}
}

func (s *showInfo) CanHandleMessage(u *tgbotapi.Update) bool {
	return u.Message != nil && (strings.HasPrefix(u.Message.Text, "/start") ||
		strings.Contains(strings.ToLower(u.Message.Text), "что ты умеешь") ||
		strings.Contains(strings.ToLower(u.Message.Text), "что ты можешь"))
}

func (s *showInfo) HandleMessage(u *tgbotapi.Update) {
	s.client.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   domain.WelcomeMessage,
	})
}
