package handler

import (
	"context"
	"time"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type TelegramClient interface {
	SendResponse(ctx context.Context, chatID int64, response *domain.Response)
	SendError(ctx context.Context, chatID int64, err error)
	SendKeyboard(ctx context.Context, chatID int64, options map[string]string, title string)
	AcknowledgeCallback(ctx context.Context, callbackQueryID string)
	DownloadFile(ctx context.Context, fileID string) ([]byte, error)
}

type ChatService interface {
	GenerateResponse(ctx context.Context, chatID int64, imageData []byte, prompt string) (*domain.Response, error)
	GenerateResponseFromVoice(ctx context.Context, chatID int64, voiceData []byte) (*domain.Response, error)
	ClearChatHistory(ctx context.Context, chatID int64) error
	GetChatSettings(ctx context.Context, chatID int64) (map[string]string, error)
	GetChatStyles(ctx context.Context, chatID int64) ([]domain.ChatStyle, error)
	SetChatTTL(ctx context.Context, chatID int64, ttl time.Duration) error
}

type ImageService interface {
	GenerateImageByID(ctx context.Context, chatID, imageID int64) (*domain.Image, error)
	SetImageStyle(ctx context.Context, chatID int64, style string) error
}
