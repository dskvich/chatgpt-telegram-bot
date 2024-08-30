package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setChatTTL struct {
	outCh chan<- domain.Message
}

func NewSetChatTTL(
	outCh chan<- domain.Message,
) *setChatTTL {
	return &setChatTTL{
		outCh: outCh,
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
	s.outCh <- &domain.TTLMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
	}
}
