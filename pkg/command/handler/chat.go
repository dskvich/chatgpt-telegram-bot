package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveMessages(chatID int64)
}

type chat struct {
	remover MessagesRemover
}

func NewChat(remover MessagesRemover) *chat {
	return &chat{remover: remover}
}

func (n *chat) CanHandle(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	text := strings.ToLower(update.Message.Text)

	if strings.HasPrefix(text, "/new_chat") ||
		strings.Contains(text, "новый чат") {
		return true
	}

	return false
}

func (n *chat) Handle(update *tgbotapi.Update) domain.Message {
	n.remover.RemoveMessages(update.Message.Chat.ID)
	return &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          "Старт нового чата. Предыдущая история беседы была очищена. Начните разговор заново.",
	}
}
