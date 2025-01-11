package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ReadSettingsRepository interface {
	GetAll(ctx context.Context, chatID int64) (map[string]string, error)
}

type showSettings struct {
	client TelegramClient
	repo   ReadSettingsRepository
}

func NewShowSettings(
	client TelegramClient,
	repo ReadSettingsRepository,
) *showSettings {
	return &showSettings{
		client: client,
		repo:   repo,
	}
}

func (s *showSettings) CanHandleMessage(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(u.Message.Text, "/config")
}

func (s *showSettings) HandleMessage(u *tgbotapi.Update) {
	settings, err := s.repo.GetAll(context.Background(), u.Message.Chat.ID)
	if err != nil {
		slog.Error("failed to get chat settings", "chatId", u.Message.Chat.ID, logger.Err(err))
	}

	message := s.formatForTelegram(settings)

	s.client.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   message,
	})
}

func (s *showSettings) formatForTelegram(data map[string]string) string {
	var sb strings.Builder
	sb.WriteString("*Системные настройки:*\n\n")
	sb.WriteString("```\n") // Code block for monospace formatting

	// Generate table headers and underline
	sb.WriteString(fmt.Sprintf("%-20s | %-30s\n", "Настройка", "Значение"))
	sb.WriteString(strings.Repeat("-", 55) + "\n")

	// Add each setting to the table
	for key, value := range data {
		sb.WriteString(fmt.Sprintf("%-20s | %-30s\n", key, value))
	}

	sb.WriteString("```\n") // End code block
	return sb.String()
}
