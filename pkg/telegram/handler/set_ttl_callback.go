package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setTTLCallback struct {
	chatService    ChatService
	telegramClient TelegramClient
	ttlOptions     map[string]time.Duration
}

func NewSetTTLCallback(
	chatService ChatService,
	telegramClient TelegramClient,
	ttlOptions map[string]time.Duration,
) *setTTLCallback {
	return &setTTLCallback{
		chatService:    chatService,
		telegramClient: telegramClient,
		ttlOptions:     ttlOptions,
	}
}

func (*setTTLCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.SetChatTTLCallbackPrefix)
}

func (s *setTTLCallback) Handle(ctx context.Context, u *tgbotapi.Update) {
	defer s.telegramClient.AcknowledgeCallback(ctx, u.CallbackQuery.ID)

	chatID := u.CallbackQuery.Message.Chat.ID
	data := u.CallbackQuery.Data

	ttl, exists := s.ttlOptions[data]
	if !exists {
		s.telegramClient.SendError(ctx, chatID, errors.New("unknown TTL option selected"))
		return
	}

	if err := s.chatService.SetChatTTL(ctx, chatID, ttl); err != nil {
		s.telegramClient.SendError(ctx, chatID, fmt.Errorf("setting chat ttl: %s", err))
		return
	}

	ttlText := "Disabled"
	if ttl > 0 {
		ttlText = ttl.String()
	}

	text := "✨TTL успешно установлен: " + ttlText
	s.telegramClient.SendResponse(ctx, chatID, &domain.Response{Text: text})
}
