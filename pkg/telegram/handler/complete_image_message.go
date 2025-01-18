package handler

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type completeImageMessage struct {
	openAiClient   OpenAiClient
	telegramClient TelegramClient
}

func NewCompleteImageMessage(
	openAiClient OpenAiClient,
	telegramClient TelegramClient,
) *completeImageMessage {
	return &completeImageMessage{
		openAiClient:   openAiClient,
		telegramClient: telegramClient,
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

	base64image, err := c.telegramClient.GetFile(photo.FileID)
	if err != nil {
		c.telegramClient.SendTextMessage(domain.TextMessage{
			ChatID: chatID,
			Text:   fmt.Sprintf("Failed to get image: %c", err),
		})
		return
	}

	response, err := c.openAiClient.CreateChatCompletion(chatID, caption, base64image)
	if err != nil {
		response = fmt.Sprintf("Failed to recognize image: %c", err)
	}

	c.telegramClient.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   response,
	})
}
