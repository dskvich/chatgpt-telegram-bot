package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatStyleDeleteRepository interface {
	DeleteAll(ctx context.Context, chatID int64) error
	DeleteByName(ctx context.Context, chatID int64, name string) error
}

type deleteChatStyle struct {
	repo ChatStyleDeleteRepository
}

func NewDeleteChatStyle(repo ChatStyleDeleteRepository) *deleteChatStyle {
	return &deleteChatStyle{
		repo: repo,
	}
}

func (a *deleteChatStyle) Name() string {
	return "delete_chat_style"
}

func (a *deleteChatStyle) Description() string {
	return "Delete a communication style by name or all styles if name is empty."
}

func (a *deleteChatStyle) Parameters() domain.Definition {
	return domain.Definition{
		Type: domain.Object,
		Properties: map[string]domain.Definition{
			"name": {
				Type:        domain.String,
				Description: "The unique name of the communication style to delete.",
			},
		},
		Required: []string{"name"},
	}
}

func (a *deleteChatStyle) Function() any {
	return func(ctx context.Context, chatID int64, name string) (string, error) {
		slog.DebugContext(ctx, "Tool invoked with args", "chatID", chatID, "name", name)

		if name != "" {
			if err := a.repo.DeleteByName(ctx, chatID, name); err != nil {
				return "", fmt.Errorf("deleting style with name '%s' for chat '%d': %w", name, chatID, err)
			}
			return fmt.Sprintf("Стиль %s успешно удален", name), nil
		}

		if err := a.repo.DeleteAll(ctx, chatID); err != nil {
			return "", fmt.Errorf("deleting all chats for chat '%d': %w", chatID, err)
		}

		return "Все стили успешно удалены", nil
	}
}
