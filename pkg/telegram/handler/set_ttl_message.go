package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setTTLMessage struct {
	client TelegramClient
}

func NewSetTTLMessage(
	client TelegramClient,
) *setTTLMessage {
	return &setTTLMessage{
		client: client,
	}
}

func (_ *setTTLMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/ttl")
}

func (s *setTTLMessage) Handle(u *tgbotapi.Update) {
	s.client.SendTTLMessage(domain.TTLMessage{
		ChatID: u.Message.Chat.ID,
	})
}
