package tools

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type EditSettingsRepository interface {
	Save(ctx context.Context, chatID int64, key, value string) error
}

type setSystemPrompt struct {
	settingsRepo EditSettingsRepository
}

func NewSetSystemPrompt(settingsRepo EditSettingsRepository) *setSystemPrompt {
	return &setSystemPrompt{
		settingsRepo: settingsRepo,
	}
}

func (s *setSystemPrompt) Name() string {
	return "set_system_prompt"
}

func (s *setSystemPrompt) Description() string {
	return "Set the system prompt to guide the assistant's behavior. This special message instructs the assistant to " +
		"follow the specified interaction style and guidelines, ensuring consistent and appropriate responses."
}

func (s *setSystemPrompt) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"prompt": {
				Type:        jsonschema.String,
				Description: "The system prompt",
			},
		},
		Required: []string{"prompt"},
	}
}

func (s *setSystemPrompt) Function() any {
	return func(chatID int64, prompt string) (string, error) {
		if err := s.settingsRepo.Save(context.Background(), chatID, domain.SystemPromptKey, prompt); err != nil {
			return "", fmt.Errorf("saving syetm prompt: %v", err)
		}
		return "Новые системные настройки сохранены", nil
	}
}
