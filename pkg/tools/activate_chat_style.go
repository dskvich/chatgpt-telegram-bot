package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatStyleActivateRepository interface {
	Activate(ctx context.Context, chatID int64, name string) error
}

type activateChatStyle struct {
	repo ChatStyleActivateRepository
}

func NewActivateChatStyle(repo ChatStyleActivateRepository) *activateChatStyle {
	return &activateChatStyle{
		repo: repo,
	}
}

func (a *activateChatStyle) Name() string {
	return "activate_chat_style"
}

func (a *activateChatStyle) Description() string {
	return "Activates a communication style by directly invoking the activation logic " +
		"based on its unique name."
}

func (a *activateChatStyle) Parameters() domain.Definition {
	return domain.Definition{
		Type: domain.Object,
		Properties: map[string]domain.Definition{
			"name": {
				Type:        domain.String,
				Description: "The unique name of the communication style to activate.",
			},
		},
		Required: []string{"name"},
	}
}

// Function provides the logic to activate a chat style by retrieving and applying it.
func (a *activateChatStyle) Function() any {
	return func(ctx context.Context, chatID int64, name string) (string, error) {
		slog.DebugContext(ctx, "Tool invoked with args", "chatID", chatID, "name", name)

		if err := a.repo.Activate(ctx, chatID, name); err != nil {
			return "", fmt.Errorf("activating chat style '%s' for chat '%d': %w", name, chatID, err)
		}

		return fmt.Sprintf("Стиль общения '%s' успешно активирован для чата '%d'", name, chatID), nil
	}
}
