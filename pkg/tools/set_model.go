package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type setModel struct {
	repo EditSettingsRepository
}

func NewSetModel(repo EditSettingsRepository) *setModel {
	return &setModel{
		repo: repo,
	}
}

func (s *setModel) Name() string {
	return "set_model"
}

func (s *setModel) Description() string {
	return "Set the current model that the assistant is using."
}

func (s *setModel) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"model": {
				Type:        jsonschema.String,
				Description: "The model name",
			},
		},
		Required: []string{"model"},
	}
}

func (s *setModel) Function() any {
	return func(chatID int64, model string) (string, error) {
		properModel, err := findModel(model)
		if err != nil {
			return "", err
		}

		if err := s.repo.Save(context.Background(), chatID, domain.ModelKey, properModel); err != nil {
			return "", fmt.Errorf("saving model: %v", err)
		}
		return "Модель установлена", nil
	}
}

func findModel(userModel string) (string, error) {
	normalizedModel := strings.ToLower(userModel)
	for _, model := range domain.SupportedModels {
		if normalizedModel == strings.ToLower(model) {
			return model, nil
		}
	}
	return "", fmt.Errorf("unsupported model: %s", userModel)
}
