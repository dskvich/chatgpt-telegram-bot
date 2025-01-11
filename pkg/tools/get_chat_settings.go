package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/logger"
)

type ChatSettingsGetRepository interface {
	GetAll(ctx context.Context, chatID int64) (map[string]string, error)
}

type getChatSettings struct {
	repo ChatSettingsGetRepository
}

func NewGetChatSettings(fetcher ChatSettingsGetRepository) *getChatSettings {
	return &getChatSettings{
		repo: fetcher,
	}
}

func (g *getChatSettings) Name() string {
	return "get_telegram_chat_settings"
}

func (g *getChatSettings) Description() string {
	return "Get telegram chat settings"
}

func (g *getChatSettings) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
	}
}

func (g *getChatSettings) Function() any {
	return func(chatID int64) (string, error) {
		settings, err := g.repo.GetAll(context.Background(), chatID)
		if err != nil {
			slog.Error("failed to get chat settings", "chatId", chatID, logger.Err(err))
			return "", err
		}

		return fmt.Sprint(settings), nil
	}
}
