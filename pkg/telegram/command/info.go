package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type info struct {
	outCh chan<- domain.Message
}

func NewInfo(
	outCh chan<- domain.Message,
) *info {
	return &info{
		outCh: outCh,
	}
}

func (_ *info) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	return strings.HasPrefix(update.Message.Text, "/start") ||
		strings.Contains(strings.ToLower(update.Message.Text), "что ты умеешь")
}

func (i *info) Execute(update *tgbotapi.Update) {
	i.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          domain.WelcomeMessage,
	}
}
