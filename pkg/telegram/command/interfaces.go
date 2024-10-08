package command

import "github.com/dskvich/chatgpt-telegram-bot/pkg/domain"

type TelegramClient interface {
	SendTextMessage(msg domain.TextMessage)
	SendImageMessage(msg domain.ImageMessage)
	SendTTLMessage(msg domain.TTLMessage)
	SendCallbackMessage(msg domain.CallbackMessage)
}
