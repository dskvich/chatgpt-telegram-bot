package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
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

func (g *getChatSettings) Parameters() domain.Definition {
	return domain.Definition{
		Type: domain.Object,
	}
}

func (g *getChatSettings) Function() any {
	return func(ctx context.Context, chatID int64) (string, error) {
		slog.DebugContext(ctx, "Tool invoked with args", "chatID", chatID)

		settings, err := g.repo.GetAll(ctx, chatID)
		if err != nil {
			return "", fmt.Errorf("getting settings for chat '%d': %w", chatID, err)
		}

		return fmt.Sprint(settings), nil
	}
}
