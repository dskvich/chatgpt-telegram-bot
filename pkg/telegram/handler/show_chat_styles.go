package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ChatStyleRepository interface {
	GetAllStyles(ctx context.Context, chatID int64) ([]domain.ChatStyle, error)
}

type showChatStyles struct {
	client TelegramClient
	repo   ChatStyleRepository
}

func NewShowChatStyles(
	client TelegramClient,
	repo ChatStyleRepository,
) *showChatStyles {
	return &showChatStyles{
		client: client,
		repo:   repo,
	}
}

func (s *showChatStyles) CanHandleMessage(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(u.Message.Text, "/styles")
}

func (s *showChatStyles) HandleMessage(u *tgbotapi.Update) {
	styles, err := s.repo.GetAllStyles(context.Background(), u.Message.Chat.ID)
	if err != nil {
		slog.Error("failed to get chat styles", "chatId", u.Message.Chat.ID, logger.Err(err))
		s.client.SendTextMessage(domain.TextMessage{
			ChatID: u.Message.Chat.ID,
			Text:   "Не удалось получить список стилей общения. Пожалуйста, попробуйте позже.",
		})
		return
	}

	message := s.formatForTelegram(styles)

	s.client.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   message,
	})
}

func (s *showChatStyles) formatForTelegram(styles []domain.ChatStyle) string {
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
