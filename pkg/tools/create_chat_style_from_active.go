package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatStyleCreateRepository interface {
	NewStyleFromActive(ctx context.Context, chatID int64, name, createdBy string) error
}

type createChatStyleFromActive struct {
	repo ChatStyleCreateRepository
}

func NewCreateChatStyleFromActive(repo ChatStyleCreateRepository) *createChatStyleFromActive {
	return &createChatStyleFromActive{
		repo: repo,
	}
}

func (c *createChatStyleFromActive) Name() string {
	return "create_chat_style_from_active"
}

func (c *createChatStyleFromActive) Description() string {
	return "Creates a new communication style based on the current active style. " +
		"Use this tool only when explicitly instructed to create a new style with a unique name. " +
		"For example, when the user specifies they want to save the current style as 'Concise and to the point.'"
}

func (c *createChatStyleFromActive) Parameters() domain.Definition {
	return domain.Definition{
		Type: domain.Object,
		Properties: map[string]domain.Definition{
			"name": {
				Type: domain.String,
				Description: "The unique name for the new communication style. " +
					"Use this when explicitly instructed to create a new style with a specific name. " +
					"For example: 'Concise and to the point.'",
			},
		},
		Required: []string{"name"},
	}
}

func (c *createChatStyleFromActive) Function() any {
	return func(ctx context.Context, chatID int64, name string) (string, error) {
		slog.DebugContext(ctx, "Tool invoked with args", "chatID", chatID, "name", name)

		if err := c.repo.NewStyleFromActive(ctx, chatID, name, "admin"); err != nil {
			return "", fmt.Errorf("creating new chat style from active for chat '%d': %w", chatID, err)
		}

		return fmt.Sprintf("Стиль общения '%s' создан.", name), nil
	}
}
