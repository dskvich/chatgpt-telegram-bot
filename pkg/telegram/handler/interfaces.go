package handler

import "github.com/dskvich/chatgpt-telegram-bot/pkg/domain"

type TelegramClient interface {
	SendTextMessage(msg domain.TextMessage)
	SendImageMessage(msg domain.ImageMessage)
	SendTTLMessage(msg domain.TTLMessage)
	SendImageStyleMessage(msg domain.TextMessage)
	SendCallbackMessage(msg domain.CallbackMessage)
	GetFile(fileID string) (string, error)
	DownloadFile(fileID string) (filePath string, err error)
}

type OpenAiClient interface {
	CreateChatCompletion(chatID int64, text, base64image string) (string, error)
	GenerateImage(chatID int64, prompt string) ([]byte, error)
	TranscribeAudio(filePath string) (string, error)
}
