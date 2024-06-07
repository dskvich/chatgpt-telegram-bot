package command

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
	response, err := g.gptProvider.GenerateChatResponse(update.Message.Chat.ID, update.Message.Text)
	if err != nil {
		response = fmt.Sprintf("Failed to get response from ChatGPT: %v", err)
	}

	g.outCh <- &domain.TextMessage{
		ChatID:           update.Message.Chat.ID,
		ReplyToMessageID: update.Message.MessageID,
		Content:          response,
	}
}
