package tools

import (
	"context"
)

type ReadSettingsRepository interface {
	GetAll(ctx context.Context, chatID int64) (map[string]string, error)
}

type EditSettingsRepository interface {
	Save(ctx context.Context, chatID int64, key, value string) error
}
