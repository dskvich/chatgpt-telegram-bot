package handler

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/domain"
)

type GptProvider interface {
	GenerateChatResponse(chatID int64, prompt string) (string, error)
}

type ActiveChatRepository interface {
	GetSession(chatID int64) (domain.ChatSession, bool)
}

type gpt struct {
	gptProvider    GptProvider
	chatRepository ActiveChatRepository
	outCh          chan<- domain.Message
}

func NewGpt(
	gptProvider GptProvider,
	chatRepository ActiveChatRepository,
	outCh chan<- domain.Message,
) *gpt {
	return &gpt{
		gptProvider:    gptProvider,
		chatRepository: chatRepository,
		outCh:          outCh,
	}
}

func (g *gpt) CanHandle(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	session, _ := g.chatRepository.GetSession(update.Message.Chat.ID)

	return update.Message.Text != "" &&
		!session.AwaitingSettings &&
		!strings.HasPrefix(update.Message.Text, "/") &&
		!strings.Contains(strings.ToLower(update.Message.Text), "рисуй")
}

func (g *gpt) Handle(update *tgbotapi.Update) {
	userName := g.getUserName(update.Message.From)
	messageToGPT := userName + " спрашивает: " + update.Message.Text

	response, err := g.gptProvider.GenerateChatResponse(update.Message.Chat.ID, messageToGPT)
	if err != nil {
		response = fmt.Sprintf("Failed to get response from ChatGPT: %v", err)
	}

	g.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}

func (g *gpt) getUserName(user *tgbotapi.User) string {
	if user.FirstName != "" {
		return user.FirstName
	}
	if user.LastName != "" {
		return user.LastName
	}
	return user.UserName
}
