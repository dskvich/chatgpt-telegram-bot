package command

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

const maxRunes = 4096

type GptProvider interface {
	CreateChatCompletion(chatID int64, text, base64image string) (string, error)
}

type ActiveChatRepository interface {
	GetSession(chatID int64) (domain.ChatSession, bool)
}

type completeChat struct {
	gptProvider    GptProvider
	chatRepository ActiveChatRepository
	client         TelegramClient
}

func NewCompleteChat(
	gptProvider GptProvider,
	chatRepository ActiveChatRepository,
	client TelegramClient,
) *completeChat {
	return &completeChat{
		gptProvider:    gptProvider,
		chatRepository: chatRepository,
		client:         client,
	}
}

func (c *completeChat) IsCommand(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	session, _ := c.chatRepository.GetSession(u.Message.Chat.ID)

	return u.Message.Text != "" &&
		!session.AwaitingSettings &&
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
