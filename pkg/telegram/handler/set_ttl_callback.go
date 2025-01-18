package handler

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type TTLSetter interface {
	SetTTL(chatID int64, ttl time.Duration)
}

type setTTLCallback struct {
	client    TelegramClient
	ttlSetter TTLSetter
}

func NewSetTTLCallback(
	client TelegramClient,
	ttlSetter TTLSetter,
) *setTTLCallback {
	return &setTTLCallback{
		client:    client,
		ttlSetter: ttlSetter,
	}
}

func (*setTTLCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.SetChatTTLCallback)
}

func (s *setTTLCallback) Handle(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	callbackQueryID := u.CallbackQuery.ID

	ttl, err := s.parseTTL(u.CallbackQuery.Data)
	if err != nil {
		s.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   "Unknown TTL option selected.",
		})
		return
	}

	s.ttlSetter.SetTTL(chatID, ttl)

	s.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: callbackQueryID,
	})

	ttlText := "disabled"
	if ttl > 0 {
		ttlText = fmt.Sprintf("%v", ttl)
	}

	s.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("Set TTL to %v", ttlText),
	})
}

func (*setTTLCallback) parseTTL(data string) (time.Duration, error) {
	switch data {
	case "ttl_15m":
		return 15 * time.Minute, nil
	case "ttl_1h":
		return time.Hour, nil
	case "ttl_8h":
		return 8 * time.Hour, nil
	case "ttl_disabled":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown TTL option")
	}
}
