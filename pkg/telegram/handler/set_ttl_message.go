package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setTTL struct {
	client TelegramClient
}

func NewSetTTL(
	client TelegramClient,
) *setTTL {
	return &setTTL{
		client: client,
	}
}

func (_ *setTTL) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/ttl")
}

func (c *setTTL) Handle(u *tgbotapi.Update) {
	c.client.SendTTLMessage(domain.TTLMessage{
		ChatID: u.Message.Chat.ID,
	})
}
