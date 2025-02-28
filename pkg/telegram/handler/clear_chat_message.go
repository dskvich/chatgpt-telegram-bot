package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type clearChatMessage struct {
	chatService    ChatService
	telegramClient TelegramClient
}

func NewClearChatMessage(
	chatService ChatService,
	telegramClient TelegramClient,
) *clearChatMessage {
	return &clearChatMessage{
		chatService:    chatService,
		telegramClient: telegramClient,
	}
}

func (*clearChatMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil &&
		(strings.ToLower(u.Message.Text) == "/new" ||
			strings.HasSuffix(strings.ToLower(u.Message.Text), "новый чат"))
}

func (c *clearChatMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID

	if err := c.chatService.ClearChatHistory(ctx, chatID); err != nil {
		c.telegramClient.SendError(ctx, chatID, fmt.Errorf("generating response: %s", err))
		return
	}

	text := "🧹 История очищена! Начните новый чат. 🚀"
	c.telegramClient.SendResponse(ctx, chatID, &domain.Response{Text: text})
}
