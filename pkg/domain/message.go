package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Message interface {
	ToChatMessage() tgbotapi.Chattable
}
