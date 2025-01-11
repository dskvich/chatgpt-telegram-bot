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

type setImageStyle struct {
	client TelegramClient
	setter ImageStyleSetter
}

func NewSetImageStyle(
	client TelegramClient,
	setter ImageStyleSetter,
) *setImageStyle {
	return &setImageStyle{
		client: client,
		setter: setter,
	}
}

func (c *setImageStyle) CanHandleMessage(u *tgbotapi.Update) bool {
	return u.Message != nil && strings.HasPrefix(strings.ToLower(u.Message.Text), "/image_style")
}

func (c *setImageStyle) HandleMessage(u *tgbotapi.Update) {
	c.client.SendImageStyleMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
	})
}

func (c *setImageStyle) CanHandleCallback(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.ImageStyleCallbackPrefix)
}

func (c *setImageStyle) HandleCallback(u *tgbotapi.Update) {
	chatID := u.CallbackQuery.Message.Chat.ID
	callbackQueryID := u.CallbackQuery.ID

	imageStyle, err := c.parseImageStyle(u.CallbackQuery.Data)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   err.Error(),
		})
		return
	}

	c.setter.Save(context.Background(), chatID, domain.ImageStyleKey, imageStyle)

	c.client.SendCallbackMessage(domain.CallbackMessage{
		CallbackQueryID: callbackQueryID,
	})

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   fmt.Sprintf("Стиль изображения успешно установлен: %v", imageStyle),
	})
}

func (c *setImageStyle) parseImageStyle(data string) (string, error) {
	if !strings.HasPrefix(data, domain.ImageStyleCallbackPrefix) {
		return "", fmt.Errorf("unknown Image Style option: invalid prefix")
	}

	key := strings.TrimPrefix(data, domain.ImageStyleCallbackPrefix)

	if _, exists := domain.ImageStyles[key]; !exists {
		return "", fmt.Errorf("unknown Image Style option: %s", key)
	}

	return key, nil
}
