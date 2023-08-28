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
	return update.Message != nil && strings.HasPrefix(update.Message.Text, "/new_chat")
}

func (n *chat) Handle(update *tgbotapi.Update) domain.Message {
	n.remover.RemoveMessages(update.Message.Chat.ID)
	return &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          "New chat created.",
	}
}
