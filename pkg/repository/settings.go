package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type settingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *settingsRepository {
	return &settingsRepository{db: db}
}

func (s *settingsRepository) Save(ctx context.Context, settings domain.Settings) error {
	const query = `
		INSERT INTO settings (chat_id, topic_id, text_model, system_prompt, image_model, ttl)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chat_id, topic_id)
		DO UPDATE SET
			text_model = EXCLUDED.text_model,
		    system_prompt = EXCLUDED.system_prompt,
			image_model = EXCLUDED.image_model,
			ttl = EXCLUDED.ttl
	`

	_, err := s.db.ExecContext(ctx, query, settings.ChatID, settings.TopicID, settings.TextModel, settings.SystemPrompt, settings.ImageModel, settings.TTL)
	if err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}

	return nil
}

func (s *settingsRepository) Get(ctx context.Context, chatID int64, topicID int) (*domain.Settings, error) {
	const query = `
		SELECT chat_id, topic_id, text_model, system_prompt, image_model, ttl
		FROM settings
		WHERE chat_id = $1
		  AND topic_id = $2
	`

	var res domain.Settings
	err := s.db.QueryRowContext(ctx, query, chatID, topicID).
		Scan(&res.ChatID, &res.TopicID, &res.TextModel, &res.SystemPrompt, &res.ImageModel, &res.TTL)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("fetching settings by chatID: %w", err)
	}

	return &res, nil
}
