package handler

import "github.com/dskvich/chatgpt-telegram-bot/pkg/domain"

type TelegramClient interface {
	SendTextMessage(msg domain.TextMessage)
	SendImageMessage(msg domain.ImageMessage)
	SendTTLMessage(msg domain.TTLMessage)
	SendImageStyleMessage(msg domain.TextMessage)
	SendCallbackMessage(msg domain.CallbackMessage)
}
