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

type gpt struct {
	gptProvider    GptProvider
	chatRepository ActiveChatRepository
	client         TelegramClient
}

func NewGpt(
	gptProvider GptProvider,
	chatRepository ActiveChatRepository,
	client TelegramClient,
) *gpt {
	return &gpt{
		gptProvider:    gptProvider,
		chatRepository: chatRepository,
		client:         client,
	}
}

func (g *gpt) CanExecute(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	session, _ := g.chatRepository.GetSession(update.Message.Chat.ID)

	return update.Message.Text != "" &&
		!session.AwaitingSettings &&
		!strings.HasPrefix(update.Message.Text, "/") &&
		!strings.Contains(strings.ToLower(update.Message.Text), "рисуй")
}

func (g *gpt) Execute(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID

	response, err := g.gptProvider.CreateChatCompletion(chatID, update.Message.Text, "")
	if err != nil {
		response = fmt.Sprintf("Failed to get chat completion: %v", err)
	}

	g.client.SendTextMessage(domain.TextMessage{
		ChatID:           chatID,
		ReplyToMessageID: messageID,
		Text:             response,
	})
}
