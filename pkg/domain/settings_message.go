package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type SettingsMessage struct {
	ChatID            int64
	ReplyToMessageID  int
	SystemPromptValue string
}

func (t *SettingsMessage) ToChatMessage() tgbotapi.Chattable {
	content := "Текущее значение: " + t.SystemPromptValue

	msg := tgbotapi.NewMessage(t.ChatID, content)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Edit", SettingsCallback),
		),
	)
	return msg
}
