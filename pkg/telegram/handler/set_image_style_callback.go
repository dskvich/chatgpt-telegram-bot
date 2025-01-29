package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setImageStyleCallback struct {
	imageService   ImageService
	telegramClient TelegramClient
	imageStyles    map[string]string
}

func NewSetImageStyleCallback(
	imageService ImageService,
	telegramClient TelegramClient,
	imageStyles map[string]string,
) *setImageStyleCallback {
	return &setImageStyleCallback{
		imageService:   imageService,
		telegramClient: telegramClient,
		imageStyles:    imageStyles,
	}
}

func (*setImageStyleCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.ImageStyleCallbackPrefix)
}

func (s *setImageStyleCallback) Handle(ctx context.Context, u *tgbotapi.Update) {
	defer s.telegramClient.AcknowledgeCallback(ctx, u.CallbackQuery.ID)

	chatID := u.CallbackQuery.Message.Chat.ID
	data := u.CallbackQuery.Data

	imageStyle, err := s.parseImageStyle(data)
	if err != nil {
		s.telegramClient.SendError(ctx, chatID, fmt.Errorf("parsing image styles: %s", err))
		return
	}

	if err := s.imageService.SetImageStyle(ctx, chatID, imageStyle); err != nil {
		s.telegramClient.SendError(ctx, chatID, fmt.Errorf("setting image style: %s", err))
		return
	}

	text := "✨Стиль изображения успешно установлен: " + imageStyle
	s.telegramClient.SendResponse(ctx, chatID, &domain.Response{Text: text})
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
