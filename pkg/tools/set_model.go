package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type SettingsRepository interface {
	Save(ctx context.Context, settings domain.Settings) error
}
type setModel struct {
	repo            SettingsRepository
	supportedModels []string
}

func NewSetModel(repo SettingsRepository, supportedModels []string) *setModel {
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

func (s *setModel) Parameters() domain.Definition {
	return domain.Definition{
		Type: domain.Object,
		Properties: map[string]domain.Definition{
			"model": {
				Type:        domain.String,
				Description: "The model name",
			},
		},
		Required: []string{"model"},
	}
}

func (s *setModel) Function() any {
	return func(ctx context.Context, chatID int64, model string) (string, error) {
		slog.DebugContext(ctx, "Tool invoked with args", "chatID", chatID, "model", model)

		properModel, err := s.findModel(model)
		if err != nil {
			return "", fmt.Errorf("looking for model '%s': %w", model, err)
		}

		settings := domain.Settings{
			ChatID:    chatID,
			TextModel: properModel,
		}

		if err := s.repo.Save(ctx, settings); err != nil {
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
