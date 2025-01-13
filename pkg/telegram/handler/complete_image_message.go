package handler

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ImageGetter interface {
	GetFile(fileID string) (base64image string, err error)
}

type ImageRecognizer interface {
	CreateChatCompletion(chatID int64, text, base64image string) (string, error)
}

type completeImageMessage struct {
	getter     ImageGetter
	recognizer ImageRecognizer
	client     TelegramClient
}

func NewCompleteImageMessage(
	getter ImageGetter,
	imageRecognizer ImageRecognizer,
	client TelegramClient,
) *completeImageMessage {
	return &completeImageMessage{
		getter:     getter,
		recognizer: imageRecognizer,
		client:     client,
	}
}

func (_ *completeImageMessage) CanHandle(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	return len(u.Message.Photo) > 0
}

func (c *completeImageMessage) Handle(u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	caption := u.Message.Caption
	photo := (u.Message.Photo)[len(u.Message.Photo)-1]

	base64image, err := c.getter.GetFile(photo.FileID)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to get image: %c", err),
		})
		return
	}

	response, err := c.recognizer.CreateChatCompletion(chatID, caption, base64image)
	if err != nil {
		response = fmt.Sprintf("Failed to recognize image: %c", err)
	}

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   response,
	})
}
