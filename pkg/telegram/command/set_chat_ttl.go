package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setChatTTL struct {
	client TelegramClient
}

func NewSetChatTTL(
	client TelegramClient,
) *setChatTTL {
	return &setChatTTL{
		client: client,
	}
}

func (s *setChatTTL) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	text := strings.ToLower(update.Message.Text)

	if strings.HasPrefix(text, "/ttl") {
		return true
	}

	return false
}

func (s *setChatTTL) Execute(update *tgbotapi.Update) {
	s.client.SendTTLMessage(domain.TTLMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
	})
}
