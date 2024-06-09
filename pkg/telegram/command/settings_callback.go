package command

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type SettingChatRepository interface {
	SaveSession(chatID int64, session domain.ChatSession)
}

type settingsCallback struct {
	repo  SettingChatRepository
	outCh chan<- domain.Message
}

func NewSettingsCallback(
	repo SettingChatRepository,
	outCh chan<- domain.Message,
) *settingsCallback {
	return &settingsCallback{
		repo:  repo,
		outCh: outCh,
	}
}

func (s *settingsCallback) CanExecute(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, domain.SettingsCallback)
}

func (s *settingsCallback) Execute(update *tgbotapi.Update) {
	chatID := update.CallbackQuery.Message.Chat.ID

	s.repo.SaveSession(chatID, domain.ChatSession{AwaitingSettings: true})

	s.outCh <- &domain.CallbackMessage{
		ID: update.CallbackQuery.ID,
	}

	s.outCh <- &domain.TextMessage{
		ChatID:  chatID,
		Content: "Введите новые настройки:",
	}
}
