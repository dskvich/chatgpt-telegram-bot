package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type MessagesRemover interface {
	RemoveSession(chatID int64)
}

type cleanChat struct {
	remover MessagesRemover
	outCh   chan<- domain.Message
}

func NewCleanChat(
	remover MessagesRemover,
	outCh chan<- domain.Message,
) *cleanChat {
	return &cleanChat{
		remover: remover,
		outCh:   outCh,
	}
}

func (c *cleanChat) CanExecute(update *tgbotapi.Update) bool {
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

func (c *cleanChat) Execute(update *tgbotapi.Update) {
	c.remover.RemoveSession(update.Message.Chat.ID)

	c.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          "Старт нового чата. Предыдущая история беседы была очищена. Начните разговор заново.",
	}
}
