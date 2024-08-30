package domain

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TTLMessage struct {
	ChatID           int64
	ReplyToMessageID int
}

func (t *TTLMessage) ToChatMessage() tgbotapi.Chattable {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("15min", "ttl_15m"),
			tgbotapi.NewInlineKeyboardButtonData("1h", "ttl_1h"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("8h", "ttl_8h"),
			tgbotapi.NewInlineKeyboardButtonData("Disabled", "ttl_disabled"),
		),
	)

	msg := tgbotapi.NewMessage(t.ChatID, "Select TTL option:")
	msg.ReplyMarkup = keyboard
	msg.ReplyToMessageID = t.ReplyToMessageID

	return msg
}
