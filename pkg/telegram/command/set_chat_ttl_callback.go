package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ChatTTLSetter interface {
	SetTTL(chatID int64, ttl time.Duration)
}

type setChatTTLCallback struct {
	ttlSetter ChatTTLSetter
	client    TelegramClient
}

func NewSetChatTTLCallback(
	ttlSetter ChatTTLSetter,
	client TelegramClient,
) *setChatTTLCallback {
	return &setChatTTLCallback{
		ttlSetter: ttlSetter,
		client:    client,
	}
}

func (s *setChatTTLCallback) CanExecute(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, domain.SetChatTTLCallback)
}

func (s *setChatTTLCallback) Execute(update *tgbotapi.Update) {
	chatID := update.CallbackQuery.Message.Chat.ID
	messageID := update.CallbackQuery.Message.ReplyToMessage.MessageID

	var ttl time.Duration

	switch update.CallbackQuery.Data {
	case "ttl_15m":
		ttl = 15 * time.Minute
	case "ttl_1h":
		ttl = time.Hour
	case "ttl_8h":
		ttl = 8 * time.Hour
	case "ttl_disabled":
	default:
		s.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             "Unknown ttl option selected.",
		})
		return
	}

	s.ttlSetter.SetTTL(chatID, ttl)

	s.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	s.client.SendTextMessage(domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Text:             fmt.Sprintf("Set TTL to %v", ttl),
	})
}
