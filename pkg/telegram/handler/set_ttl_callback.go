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
	client     TelegramClient
	ttlSetter  TTLSetter
	ttlOptions map[string]time.Duration
}

func NewSetTTLCallback(
	client TelegramClient,
	ttlSetter TTLSetter,
	ttlOptions map[string]time.Duration,
) *setTTLCallback {
	return &setTTLCallback{
		client:     client,
		ttlSetter:  ttlSetter,
		ttlOptions: ttlOptions,
	}
}

func (*setTTLCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.SetChatTTLCallback)
}

func (s *setTTLCallback) Handle(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	callbackQueryID := u.CallbackQuery.ID

	ttl, exists := s.ttlOptions[u.CallbackQuery.Data]
	if !exists {
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
		ttlText = ttl.String()
	}

	s.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("Set TTL to %v", ttlText),
	})
}
