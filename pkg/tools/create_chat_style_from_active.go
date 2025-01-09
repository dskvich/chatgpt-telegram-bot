package tools

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai/jsonschema"
)

type CreateChatStyleFromActiveRepository interface {
	NewStyleFromActive(ctx context.Context, chatID int64, name, createdBy string) error
}

type createChatStyleFromActive struct {
	styleRepo CreateChatStyleFromActiveRepository
}

func NewCreateChatStyleFromActive(styleRepo CreateChatStyleFromActiveRepository) *createChatStyleFromActive {
	return &createChatStyleFromActive{
		styleRepo: styleRepo,
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

func (c *createChatStyleFromActive) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type: jsonschema.String,
				Description: "The unique name for the new communication style. " +
					"Use this when explicitly instructed to create a new style with a specific name. " +
					"For example: 'Concise and to the point.'",
			},
		},
		Required: []string{"name"},
	}
}

func (c *createChatStyleFromActive) Function() any {
	return func(chatID int64, name string) (string, error) {
		if err := c.styleRepo.NewStyleFromActive(context.Background(), chatID, name, "admin"); err != nil {
			return "", fmt.Errorf("creating new chat style from active for chat '%d': %v", chatID, err)
		}
		return fmt.Sprintf("Стиль общения '%s' создан.", name), nil
	}
}
