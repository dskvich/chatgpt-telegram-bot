package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatRepository interface {
	ClearChat(chatID int64)
}

type clearChatMessage struct {
	repo   ChatRepository
	client TelegramClient
}

func NewClearChatMessage(repo ChatRepository, client TelegramClient) *clearChatMessage {
	return &clearChatMessage{
		repo:   repo,
		client: client,
	}
}
func (*clearChatMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/new")
}

func (c *clearChatMessage) Handle(u *tgbotapi.Update) {
	c.repo.ClearChat(u.Message.Chat.ID)

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   "История очищена. Начните новый чат.",
	})
}
