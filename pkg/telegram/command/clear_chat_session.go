package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveSession(chatID int64)
}

type cleanChatSession struct {
	remover MessagesRemover
	outCh   chan<- domain.Message
}

func NewCleanChatSession(
	remover MessagesRemover,
	outCh chan<- domain.Message,
) *cleanChatSession {
	return &cleanChatSession{
		remover: remover,
		outCh:   outCh,
	}
}

func (c *cleanChatSession) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	text := strings.ToLower(update.Message.Text)

	if strings.HasPrefix(text, "/new_chat") {
		return true
	}

	return false
}

func (c *cleanChatSession) Execute(update *tgbotapi.Update) {
	c.remover.RemoveSession(update.Message.Chat.ID)

	c.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          "История очищена. Начните новый чат.",
	}
}
