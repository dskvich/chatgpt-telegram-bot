package tools

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai/jsonschema"
)

type ChatStyleActivateRepository interface {
	Activate(ctx context.Context, chatID int64, name string) error
}

type activateChatStyle struct {
	styleRepo ChatStyleActivateRepository
}

func NewActivateChatStyle(styleRepo ChatStyleActivateRepository) *activateChatStyle {
	return &activateChatStyle{
		styleRepo: styleRepo,
	}
}

func (a *activateChatStyle) Name() string {
	return "activate_chat_style"
}

func (a *activateChatStyle) Description() string {
	return "Activates a communication style by directly invoking the activation logic " +
		"based on its unique name."
}

func (a *activateChatStyle) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type:        jsonschema.String,
				Description: "The unique name of the communication style to activate.",
			},
		},
		Required: []string{"name"},
	}
}

// Function provides the logic to activate a chat style by retrieving and applying it.
func (a *activateChatStyle) Function() any {
	return func(chatID int64, name string) (string, error) {
		if err := a.styleRepo.Activate(context.Background(), chatID, name); err != nil {
			return "", fmt.Errorf("activating chat style '%s' for chat '%d': %v", name, chatID, err)
		}

		return fmt.Sprintf("Стиль общения '%s' успешно активирован для чата '%d'", name, chatID), nil
	}
}
