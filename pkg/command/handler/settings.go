package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type SettingsRepository interface {
	GetSetting(ctx context.Context, chatID int64, key string) (string, error)
}

type settings struct {
	repo  SettingsRepository
	outCh chan<- domain.Message
}

func NewSettings(
	repo SettingsRepository,
	outCh chan<- domain.Message,
) *settings {
	return &settings{
		repo:  repo,
		outCh: outCh,
	}
}

func (s *settings) CanHandle(update *tgbotapi.Update) bool {
	return update.Message != nil && strings.HasPrefix(update.Message.Text, "/settings")
}

func (s *settings) Handle(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID

	systemPromptValue, err := s.repo.GetSetting(context.TODO(), chatID, domain.SystemPromptKey)
	if err != nil {
		s.outCh <- &domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Content:          fmt.Sprintf("Failed to fetch setting: %v", err),
		}
		return
	}

	s.outCh <- &domain.SettingsMessage{
		ChatID:            chatID,
		ReplyToMessageID:  messageID,
		SystemPromptValue: systemPromptValue,
	}
}
