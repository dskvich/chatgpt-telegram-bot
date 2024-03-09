package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type SettingsMessage struct {
	ChatID            int64
	ReplyToMessageID  int
	SystemPromptValue string
}

func (t *SettingsMessage) ToChatMessage() tgbotapi.Chattable {
	var content string
	if t.SystemPromptValue == "" {
		content = "Системные настройки не заданы. Желаете внести изменения?"
	} else {
		content = "Вот текущие системные настройки: \n\n" +
			"_" + t.SystemPromptValue + "_" +
			"\n\nЖелаете внести изменения?"
	}

	msg := tgbotapi.NewMessage(t.ChatID, content)
	msg.ParseMode = string(Markdown)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Изменить", SettingsCallback),
		),
	)
	return msg
}
