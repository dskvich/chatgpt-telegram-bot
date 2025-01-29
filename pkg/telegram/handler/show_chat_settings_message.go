package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type showChatSettingsMessage struct {
	chatService    ChatService
	telegramClient TelegramClient
}

func NewShowChatSettingsMessage(
	chatService ChatService,
	telegramClient TelegramClient,
) *showChatSettingsMessage {
	return &showChatSettingsMessage{
		chatService:    chatService,
		telegramClient: telegramClient,
	}
}

func (*showChatSettingsMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(u.Message.Text, "/config")
}

func (s *showChatSettingsMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID

	settings, err := s.chatService.GetChatSettings(ctx, chatID)
	if err != nil {
		s.telegramClient.SendError(ctx, chatID, fmt.Errorf("gettting chat settings: %s", err))
		return
	}

	text := s.formatForTelegram(settings)
	s.telegramClient.SendResponse(ctx, chatID, &domain.Response{Text: text})
}

func (*showChatSettingsMessage) formatForTelegram(data map[string]string) string {
	const tableWidth = 55

	var sb strings.Builder
	sb.WriteString("*Системные настройки:*\n\n")
	sb.WriteString("```\n") // Code block for monospace formatting

	// Generate table headers and underline
	sb.WriteString(fmt.Sprintf("%-20s | %-30s\n", "Настройка", "Значение"))
	sb.WriteString(strings.Repeat("-", tableWidth) + "\n")

	// Add each setting to the table
	for key, value := range data {
		sb.WriteString(fmt.Sprintf("%-20s | %-30s\n", key, value))
	}

	sb.WriteString("```\n") // End code block
	return sb.String()
}
