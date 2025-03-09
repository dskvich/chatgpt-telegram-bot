package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type promptsRepository struct {
	db *sql.DB
}

func NewPromptsRepository(db *sql.DB) *promptsRepository {
	return &promptsRepository{db: db}
}

func (p *promptsRepository) Save(ctx context.Context, prompt string) (int64, error) {
	const query = `
		INSERT INTO prompts (prompt) 
		VALUES ($1) 
		RETURNING id
	`

	var id int64
	if err := p.db.QueryRowContext(ctx, query, prompt).Scan(&id); err != nil {
		return 0, fmt.Errorf("saving prompt: %w", err)
	}

	return id, nil
}

func (p *promptsRepository) GetByID(ctx context.Context, id int64) (string, error) {
	const query = `
		SELECT prompt 
		FROM prompts 
		WHERE id = $1
	`

	var prompt string
	err := p.db.QueryRowContext(ctx, query, id).Scan(&prompt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("prompt not found for id %d: %w", id, err)
		}
		return "", fmt.Errorf("fetching prompt by id: %w", err)
	}

	return prompt, nil
}
