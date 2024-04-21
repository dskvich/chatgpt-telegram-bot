package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type TextMessage struct {
	ChatID           int64
	ReplyToMessageID int
	Content          string
}

func (t *TextMessage) ToChatMessage() tgbotapi.Chattable {
	msg := tgbotapi.NewMessage(t.ChatID, t.Content)
	msg.ReplyToMessageID = t.ReplyToMessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
	return msg
}
