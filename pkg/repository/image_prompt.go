package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type imagePromptRepository struct {
	db *sql.DB
}

func NewImagePromptRepository(db *sql.DB) *imagePromptRepository {
	return &imagePromptRepository{db: db}
}

func (repo *imagePromptRepository) Save(ctx context.Context, i *domain.ImagePrompt) (int64, error) {
	q := "INSERT INTO image_prompts (prompt, created_by) VALUES ($1, $2) RETURNING image_id"

	var imageID int64
	if err := repo.db.QueryRowContext(ctx, q, i.Prompt, i.FromUser).Scan(&imageID); err != nil {
		return 0, fmt.Errorf("saving image prompt: %w", err)
	}

	return imageID, nil
}

func (repo *imagePromptRepository) GetPrompt(ctx context.Context, imageID int64) (string, error) {
	q := "SELECT prompt FROM image_prompts WHERE image_id = $1"

	var prompt string
	if err := repo.db.QueryRowContext(ctx, q, imageID).Scan(&prompt); err != nil {
		return "", fmt.Errorf("fetching prompt: %w", err)
	}

	return prompt, nil
}
