package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type generateImageCallback struct {
	imageService   ImageService
	telegramClient TelegramClient
}

func NewGenerateImageCallback(
	imageService ImageService,
	telegramClient TelegramClient,
) *generateImageCallback {
	return &generateImageCallback{
		imageService:   imageService,
		telegramClient: telegramClient,
	}
}

func (*generateImageCallback) CanHandle(u *tgbotapi.Update) bool {
	return u.CallbackQuery != nil && strings.HasPrefix(u.CallbackQuery.Data, domain.GenImageCallbackPrefix)
}

func (g *generateImageCallback) Handle(ctx context.Context, u *tgbotapi.Update) {
	defer g.telegramClient.AcknowledgeCallback(ctx, u.CallbackQuery.ID)

	chatID := u.CallbackQuery.Message.Chat.ID

	imageID, err := g.parseImageID(u.CallbackQuery.Data)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("parsing image ID: %s", err))
		return
	}

	slog.InfoContext(ctx, "ImageID parsed successfully", "imageID", imageID)

	image, err := g.imageService.GenerateImageByID(ctx, chatID, imageID)
	if err != nil {
		g.telegramClient.SendError(ctx, chatID, fmt.Errorf("generating image: %s", err))
		return
	}

	g.telegramClient.SendResponse(ctx, chatID, &domain.Response{Image: image})
}

func (g *generateImageCallback) parseImageID(callback string) (int64, error) {
	idStr := strings.TrimPrefix(callback, domain.GenImageCallbackPrefix)

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid image ID: %s", idStr)
	}

	return id, nil
}
