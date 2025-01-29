package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type generateResponseFromVoiceMessage struct {
	chatService    ChatService
	telegramClient TelegramClient
}

func NewGenerateResponseFromVoiceMessage(
	chatService ChatService,
	telegramClient TelegramClient,
) *generateResponseFromVoiceMessage {
	return &generateResponseFromVoiceMessage{
		chatService:    chatService,
		telegramClient: telegramClient,
	}
}

func (*generateResponseFromVoiceMessage) CanHandle(u *tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Voice != nil
}

func (g *generateResponseFromVoiceMessage) Handle(ctx context.Context, u *tgbotapi.Update) {
	chatID := u.Message.Chat.ID
	voiceFileID := u.Message.Voice.FileID

	voiceData, err := g.telegramClient.DownloadFile(ctx, voiceFileID)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("downloading voice file: %s", err))
		return
	}

	response, err := g.chatService.GenerateResponseFromVoice(ctx, chatID, voiceData)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("generating response from voice: %s", err))
		return
	}

	g.telegramClient.SendResponse(ctx, chatID, response)
}
