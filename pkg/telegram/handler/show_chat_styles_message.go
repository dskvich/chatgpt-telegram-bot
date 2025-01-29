package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type showChatStylesMessage struct {
	chatService    ChatService
	telegramClient TelegramClient
}

func NewShowChatStylesMessage(
	chatService ChatService,
	telegramClient TelegramClient,
) *showChatStylesMessage {
	return &showChatStylesMessage{
		chatService:    chatService,
		telegramClient: telegramClient,
	}
}

func (*showChatStylesMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(u.Message.Text, "/styles")
}

func (s *showChatStylesMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID

	styles, err := s.chatService.GetChatStyles(ctx, chatID)
	if err != nil {
		s.telegramClient.SendError(ctx, chatID, fmt.Errorf("getting chat styles: %s", err))
		return
	}

	text := s.formatForTelegram(styles)
	s.telegramClient.SendResponse(ctx, chatID, &domain.Response{Text: text})
}

func (*showChatStylesMessage) formatForTelegram(styles []domain.ChatStyle) string {
	if len(styles) == 0 {
		return "Нет доступных стилей общения для данного чата."
	}

	var sb strings.Builder
	sb.WriteString("*Доступные стили общения:*\n\n")

	for _, style := range styles {
		isActive := "Да"
		if !style.IsActive {
			isActive = "Нет"
		}
		sb.WriteString(fmt.Sprintf(
			"Имя стиля: %s\nОписание: %s\nАктивный: %s\n\n",
			style.Name, style.Description, isActive,
		))
	}

	return sb.String()
}
