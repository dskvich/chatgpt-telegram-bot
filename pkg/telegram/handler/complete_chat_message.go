package handler

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type completeChatMessage struct {
	openAiClient   OpenAiClient
	telegramClient TelegramClient
}

func NewCompleteChatMessage(
	openAiClient OpenAiClient,
	telegramClient TelegramClient,
) *completeChatMessage {
	return &completeChatMessage{
		openAiClient:   openAiClient,
		telegramClient: telegramClient,
	}
}

func (_ *completeChatMessage) CanHandle(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	return u.Message.Text != "" &&
		!strings.HasPrefix(u.Message.Text, "/") &&
		!strings.Contains(strings.ToLower(u.Message.Text), "рисуй")
}

func (c *completeChatMessage) Handle(u *tgbotapi.Update) {
	response, err := c.openAiClient.CreateChatCompletion(u.Message.Chat.ID, u.Message.Text, "")
	if err != nil {
		response = fmt.Sprintf("Failed to get chat completion: %v", err)
	}

	c.telegramClient.SendTextMessage(domain.TextMessage{
		ChatID: u.Message.Chat.ID,
		Text:   response,
	})
}
