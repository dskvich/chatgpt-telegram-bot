package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveSession(chatID int64)
}

type clearSession struct {
	remover MessagesRemover
	outCh   chan<- domain.Message
}

func NewClearSession(
	remover MessagesRemover,
	outCh chan<- domain.Message,
) *clearSession {
	return &clearSession{
		remover: remover,
		outCh:   outCh,
	}
}

func (c *clearSession) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	text := strings.ToLower(update.Message.Text)

	if strings.HasPrefix(text, "/new") {
		return true
	}

	return false
}

func (c *clearSession) Execute(update *tgbotapi.Update) {
	c.remover.RemoveSession(update.Message.Chat.ID)

	c.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          "История очищена. Начните новый чат.",
	}
}
