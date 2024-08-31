package command

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

type completeImage struct {
	getter     ImageGetter
	recognizer ImageRecognizer
	client     TelegramClient
}

func NewCompleteImage(
	getter ImageGetter,
	imageRecognizer ImageRecognizer,
	client TelegramClient,
) *completeImage {
	return &completeImage{
		getter:     getter,
		recognizer: imageRecognizer,
		client:     client,
	}
}

func (c *completeImage) IsCommand(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	return len(u.Message.Photo) > 0
}

func (c *completeImage) HandleCommand(u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	messageID := u.Message.MessageID
	caption := u.Message.Caption
	photo := (u.Message.Photo)[len(u.Message.Photo)-1]

	base64image, err := c.getter.GetFile(photo.FileID)
	if err != nil {
		c.client.SendTextMessage(domain.TextMessage{
			ChatID:           chatID,
			ReplyToMessageID: messageID,
			Text:             fmt.Sprintf("Failed to get image: %c", err),
		})
		return
	}

	response, err := c.recognizer.CreateChatCompletion(chatID, caption, base64image)
	if err != nil {
		response = fmt.Sprintf("Failed to recognize image: %c", err)
	}

	c.client.SendTextMessage(domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Text:             response,
	})
}
