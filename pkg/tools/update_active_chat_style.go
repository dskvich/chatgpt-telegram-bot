package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatStyleUpdateRepository interface {
	UpdateActiveStyle(ctx context.Context, chatID int64, newInstruction string) error
}

type updateActiveChatStyle struct {
	repo ChatStyleUpdateRepository
}

func NewUpdateActiveChatStyle(repo ChatStyleUpdateRepository) *updateActiveChatStyle {
	return &updateActiveChatStyle{
		repo: repo,
	}
}

func (u *updateActiveChatStyle) Name() string {
	return "update_active_chat_style"
}

func (u *updateActiveChatStyle) Description() string {
	return "Adds new instruction to the current communication style. " +
		"Use this tool to refine or adjust the style without replacing it entirely. " +
		"For instance, if the current style is 'Speak like an engineer,' and the user adds 'Make it more emotional,' " +
		"this tool will incorporate the new instruction into the existing style description."
}

func (u *updateActiveChatStyle) Parameters() domain.Definition {
	return domain.Definition{
		Type: domain.Object,
		Properties: map[string]domain.Definition{
			"newInstruction": {
				Type: domain.String,
				Description: "A new instruction to enhance the current communication style. " +
					"For example: 'Add a touch of humor.' This instruction will be appended to the existing description.",
			},
		},
		Required: []string{"newInstruction"},
	}
}

func (u *updateActiveChatStyle) Function() any {
	return func(ctx context.Context, chatID int64, newInstruction string) (string, error) {
		slog.DebugContext(ctx, "Tool invoked with args", "chatID", chatID, "newInstruction", newInstruction)

		if err := u.repo.UpdateActiveStyle(ctx, chatID, newInstruction); err != nil {
			return "", fmt.Errorf("updating active chat style for chat '%d': %w", chatID, err)
		}

		return "Стиль общения обновлен", nil
	}
}
