package command

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type GptProvider interface {
	CreateChatCompletion(chatID int64, text, base64image string) (string, error)
}

type completeChat struct {
	gptProvider GptProvider
	client      TelegramClient
}

func NewCompleteChat(
	gptProvider GptProvider,
	client TelegramClient,
) *completeChat {
	return &completeChat{
		gptProvider: gptProvider,
		client:      client,
	}
}

func (c *completeChat) IsCommand(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	return u.Message.Text != "" &&
		!strings.HasPrefix(u.Message.Text, "/") &&
		!strings.Contains(strings.ToLower(u.Message.Text), "рисуй")
}

func (c *completeChat) HandleCommand(u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID

	response, err := c.gptProvider.CreateChatCompletion(chatID, u.Message.Text, "")
	if err != nil {
		response = fmt.Sprintf("Failed to get chat completion: %v", err)
	}

	c.client.SendTextMessage(domain.TextMessage{
		ChatID: chatID,
		Text:   response,
	})
}
