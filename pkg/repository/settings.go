package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type settingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *settingsRepository {
	return &settingsRepository{db: db}
}

func (repo *settingsRepository) Save(ctx context.Context, chatID int64, key, value string) error {
	args := []any{chatID, key, value}
	query := `INSERT INTO settings (chat_id, key, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, key)
		DO UPDATE SET value = EXCLUDED.value;`

	if _, err := repo.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("saving setting: %w", err)
	}

	return nil
}

func (repo *settingsRepository) GetByKey(ctx context.Context, chatID int64, key string) (string, error) {
	query := "SELECT value FROM settings WHERE chat_id = $1 AND key = $2"

	var value string
	if err := repo.db.QueryRowContext(ctx, query, chatID, key).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // Return an empty string with no error
		}
		return "", fmt.Errorf("fetching setting value: %w", err)
	}
	return value, nil
}

func (repo *settingsRepository) GetAll(ctx context.Context, chatID int64) (map[string]string, error) {
	query := "SELECT key, value FROM settings WHERE chat_id = $1"

	rows, err := repo.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("fetching all settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning setting row: %w", err)
		}
		settings[key] = value
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating setting rows: %w", err)
	}

	return settings, nil
}
