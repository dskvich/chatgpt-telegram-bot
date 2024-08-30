package domain

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TypingMessage struct {
	ChatID int64
}

func (t *TypingMessage) ToChatMessage() tgbotapi.Chattable {
	return tgbotapi.NewChatAction(t.ChatID, tgbotapi.ChatTyping)
}
