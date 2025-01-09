package tools

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai/jsonschema"
)

type UpdateActiveChatStyleRepository interface {
	UpdateActiveStyle(ctx context.Context, chatID int64, description string) error
}

type updateActiveChatStyle struct {
	styleRepo UpdateActiveChatStyleRepository
}

func NewUpdateActiveChatStyle(styleRepo UpdateActiveChatStyleRepository) *updateActiveChatStyle {
	return &updateActiveChatStyle{
		styleRepo: styleRepo,
	}
}

func (u *updateActiveChatStyle) Name() string {
	return "update_active_chat_style"
}

func (u *updateActiveChatStyle) Description() string {
	return "Updates the currently active communication style by modifying its description. " +
		"Use this tool when the user provides additional details or changes to the current style without naming a new one. " +
		"For example, when the user says, 'Speak like a geek.'"
}

func (u *updateActiveChatStyle) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"description": {
				Type: jsonschema.String,
				Description: "The updated description for the currently active communication style. " +
					"Use this when modifying or adding details to the active style without creating a new one. " +
					"For example: 'Speak like a geek.'",
			},
		},
		Required: []string{"description"},
	}
}

func (u *updateActiveChatStyle) Function() any {
	return func(chatID int64, description string) (string, error) {
		if err := u.styleRepo.UpdateActiveStyle(context.Background(), chatID, description); err != nil {
			return "", fmt.Errorf("updating active chat style for chat '%d': %v", chatID, err)
		}
		return "Стиль общения обновлен", nil
	}
}
