package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type ImageMessage struct {
	ChatID           int64
	ReplyToMessageID int
	Prompt           string
	Content          []byte
}

func (i *ImageMessage) ToChatMessage() tgbotapi.Chattable {
	fileBytes := tgbotapi.FileBytes{
		Bytes: i.Content,
	}
	msg := tgbotapi.NewPhoto(i.ChatID, fileBytes)
	msg.ReplyToMessageID = i.ReplyToMessageID

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Еще", DrawCallback),
		),
	)
	msg.ReplyMarkup = keyboard

	return msg
}
