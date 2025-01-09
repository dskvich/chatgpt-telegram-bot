package tools

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai/jsonschema"
)

type UpdateActiveChatStyleRepository interface {
	UpdateActiveStyle(ctx context.Context, chatID int64, newInstruction string) error
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
	return "Adds new instruction to the current communication style. " +
		"Use this tool to refine or adjust the style without replacing it entirely. " +
		"For instance, if the current style is 'Speak like an engineer,' and the user adds 'Make it more emotional,' " +
		"this tool will incorporate the new instruction into the existing style description."
}

func (u *updateActiveChatStyle) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"newInstruction": {
				Type: jsonschema.String,
				Description: "A new instruction to enhance the current communication style. " +
					"For example: 'Add a touch of humor.' This instruction will be appended to the existing description.",
			},
		},
		Required: []string{"newInstruction"},
	}
}

func (u *updateActiveChatStyle) Function() any {
	return func(chatID int64, newInstruction string) (string, error) {
		if err := u.styleRepo.UpdateActiveStyle(context.Background(), chatID, newInstruction); err != nil {
			return "", fmt.Errorf("updating active chat style for chat '%d': %v", chatID, err)
		}
		return "Стиль общения обновлен", nil
	}
}
