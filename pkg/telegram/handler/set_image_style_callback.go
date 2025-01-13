package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ImageStyleSetter interface {
	Save(ctx context.Context, chatID int64, key, value string) error
}

type setImageStyleCallback struct {
	client TelegramClient
	setter ImageStyleSetter
}

func NewSetImageStyleCallback(
	client TelegramClient,
	setter ImageStyleSetter,
) *setImageStyleCallback {
	return &setImageStyleCallback{
		client: client,
		setter: setter,
	}
}

func (_ *setImageStyleCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.ImageStyleCallbackPrefix)
}

func (s *setImageStyleCallback) Handle(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	callbackQueryID := u.CallbackQuery.ID

	imageStyle, err := s.parseImageStyle(u.CallbackQuery.Data)
	if err != nil {
		s.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   err.Error(),
		})
		return
	}

	s.setter.Save(context.Background(), chatID, domain.ImageStyleKey, imageStyle)

	s.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: callbackQueryID,
	})

	s.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("Стиль изображения успешно установлен: %v", imageStyle),
	})
}

func (_ *setImageStyleCallback) parseImageStyle(data string) (string, error) {
	if !strings.HasPrefix(data, domain.ImageStyleCallbackPrefix) {
		return "", fmt.Errorf("unknown Image Style option: invalid prefix")
	}

	key := strings.TrimPrefix(data, domain.ImageStyleCallbackPrefix)

	if _, exists := domain.ImageStyles[key]; !exists {
		return "", fmt.Errorf("unknown Image Style option: %s", key)
	}

	return key, nil
}
