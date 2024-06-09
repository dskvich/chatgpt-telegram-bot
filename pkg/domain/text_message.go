package domain

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/render"
)

type TextMessage struct {
	ChatID           int64
	ReplyToMessageID int
	Content          string
}

func (t *TextMessage) ToChatMessage() tgbotapi.Chattable {
	htmlOutput := render.ToHTML(t.Content)

	msg := tgbotapi.NewMessage(t.ChatID, htmlOutput)
	msg.ReplyToMessageID = t.ReplyToMessageID
	msg.ParseMode = tgbotapi.ModeHTML

	return msg
}
