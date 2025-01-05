package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveSession(chatID int64)
}

type clearChat struct {
	remover MessagesRemover
	client  TelegramClient
}

func NewClearChat(remover MessagesRemover, client TelegramClient) *clearChat {
	return &clearChat{
		remover: remover,
		client:  client,
	}
}
func (c *clearChat) IsCommand(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/new")
}

func (c *clearChat) HandleCommand(u *tgbotapi.Update) {
	c.remover.RemoveSession(u.Message.Chat.ID)

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   "История очищена. Начните новый чат.",
	})
}
