package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type CallbackMessage struct {
	ID string
}

func (c *CallbackMessage) ToChatMessage() tgbotapi.Chattable {
	return tgbotapi.NewCallback(c.ID, "")
}
