package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type info struct {
	client TelegramClient
}

func NewInfo(
	client TelegramClient,
) *info {
	return &info{
		client: client,
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
	i.client.SendTextMessage(domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Text:             domain.WelcomeMessage,
	})
}
