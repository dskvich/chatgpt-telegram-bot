package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type ChatSettingsSaveRepository interface {
	Save(ctx context.Context, chatID int64, key, value string) error
}
type setModel struct {
	repo            ChatSettingsSaveRepository
	supportedModels []string
}

func NewSetModel(repo ChatSettingsSaveRepository, supportedModels []string) *setModel {
	return &setModel{
		repo:            repo,
		supportedModels: supportedModels,
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
		properModel, err := s.findModel(model)
		if err != nil {
			return "", err
		}

		if err := s.repo.Save(context.Background(), chatID, domain.ModelKey, properModel); err != nil {
			return "", fmt.Errorf("saving model: %w", err)
		}
		return "Модель установлена", nil
	}
}

func (s *setModel) findModel(userModel string) (string, error) {
	normalizedModel := strings.ToLower(userModel)
	for _, model := range s.supportedModels {
		if strings.EqualFold(normalizedModel, model) {
			return model, nil
		}
	}
	return "", fmt.Errorf("unsupported model: %s", userModel)
}
