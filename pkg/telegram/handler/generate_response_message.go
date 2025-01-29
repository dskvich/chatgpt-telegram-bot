package handler

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type generateResponseMessage struct {
	chatService    ChatService
	telegramClient TelegramClient
}

func NewGenerateResponseMessage(
	chatService ChatService,
	telegramClient TelegramClient,
) *generateResponseMessage {
	return &generateResponseMessage{
		chatService:    chatService,
		telegramClient: telegramClient,
	}
}

func (*generateResponseMessage) CanHandle(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	return u.Message.Text != "" && !strings.HasPrefix(u.Message.Text, "/")
}

func (g *generateResponseMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	prompt := u.Message.Text

	response, err := g.chatService.GenerateResponse(ctx, chatID, nil, prompt)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("generating response: %s", err))
		return
	}

	g.telegramClient.SendResponse(ctx, chatID, response)
}
