package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ImageStyleSetter interface {
	Save(ctx context.Context, chatID int64, key, value string) error
}

type setImageStyleCallback struct {
	client      TelegramClient
	setter      ImageStyleSetter
	imageStyles map[string]string
}

func NewSetImageStyleCallback(
	client TelegramClient,
	setter ImageStyleSetter,
	imageStyles map[string]string,
) *setImageStyleCallback {
	return &setImageStyleCallback{
		client:      client,
		setter:      setter,
		imageStyles: imageStyles,
	}
}

func (*setImageStyleCallback) CanHandle(u *tgbotapi.Update) bool {
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

	err = s.setter.Save(context.Background(), chatID, domain.ImageStyleKey, imageStyle)
	if err != nil {
		s.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   err.Error(),
		})
		return
	}

	s.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: callbackQueryID,
	})

	s.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("Стиль изображения успешно установлен: %v", imageStyle),
	})
}

func (s *setImageStyleCallback) parseImageStyle(data string) (string, error) {
	if !strings.HasPrefix(data, domain.ImageStyleCallbackPrefix) {
		return "", errors.New("unknown Image Style option: invalid prefix")
	}

	key := strings.TrimPrefix(data, domain.ImageStyleCallbackPrefix)

	if _, exists := s.imageStyles[key]; !exists {
		return "", fmt.Errorf("unknown Image Style option: %s", key)
	}

	return key, nil
}
