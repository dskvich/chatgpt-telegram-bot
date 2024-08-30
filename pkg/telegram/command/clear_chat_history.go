package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveSession(chatID int64)
}

type clearChatHistory struct {
	remover MessagesRemover
	outCh   chan<- domain.Message
}

func NewClearChatHistory(
	remover MessagesRemover,
	outCh chan<- domain.Message,
) *clearChatHistory {
	return &clearChatHistory{
		remover: remover,
		outCh:   outCh,
	}
}

func (c *clearChatHistory) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	text := strings.ToLower(update.Message.Text)

	if strings.HasPrefix(text, "/new") {
		return true
	}

	return false
}

func (c *clearChatHistory) Execute(update *tgbotapi.Update) {
	c.remover.RemoveSession(update.Message.Chat.ID)

	c.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          "История очищена. Начните новый чат.",
	}
}
