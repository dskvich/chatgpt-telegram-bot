package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type generateResponseFromImageMessage struct {
	chatService    ChatService
	telegramClient TelegramClient
}

func NewGenerateResponseFromImageMessage(
	chatService ChatService,
	telegramClient TelegramClient,
) *generateResponseFromImageMessage {
	return &generateResponseFromImageMessage{
		chatService:    chatService,
		telegramClient: telegramClient,
	}
}

func (*generateResponseFromImageMessage) CanHandle(u *tgbotapi.Update) bool {
	if u.Message == nil {
		return false
	}

	return len(u.Message.Photo) > 0
}

func (g *generateResponseFromImageMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	caption := u.Message.Caption
	photo := u.Message.Photo[len(u.Message.Photo)-1]

	imageData, err := g.telegramClient.DownloadFile(ctx, photo.FileID)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("downloading image file: %s", err))
		return
	}

	response, err := g.chatService.GenerateResponse(ctx, chatID, imageData, caption)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("generating response from image: %s", err))
		return
	}

	g.telegramClient.SendResponse(ctx, chatID, response)
}
