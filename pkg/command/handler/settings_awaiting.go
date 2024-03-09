package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type AwaitingChatRepository interface {
	GetSession(chatID int64) (domain.ChatSession, bool)
	RemoveSession(chatID int64)
}

type EditSettingsRepository interface {
	SaveSetting(ctx context.Context, chatID int64, key, value string) error
}

type settingsAwaiting struct {
	chatRepo     AwaitingChatRepository
	settingsRepo EditSettingsRepository
	outCh        chan<- domain.Message
}

func NewSettingsAwaiting(
	chatRepository AwaitingChatRepository,
	settingsRepo EditSettingsRepository,
	outCh chan<- domain.Message,
) *settingsAwaiting {
	return &settingsAwaiting{
		chatRepo:     chatRepository,
		settingsRepo: settingsRepo,
		outCh:        outCh,
	}
}

func (s *settingsAwaiting) CanHandle(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	session, _ := s.chatRepo.GetSession(update.Message.Chat.ID)

	return session.AwaitingSettings
}

func (s *settingsAwaiting) Handle(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID

	response := "Новые системные настройки сохранены"
	if err := s.settingsRepo.SaveSetting(context.TODO(), chatID, domain.SystemPromptKey, update.Message.Text); err != nil {
		response = fmt.Sprintf("Failed to fetch setting: %v", err)
	}

	// Clear chat history with old system prompt and awaiting flag
	s.chatRepo.RemoveSession(chatID)

	s.outCh <- &domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}
