package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type promptRepository struct {
	db *sql.DB
}

func NewPromptRepository(db *sql.DB) *promptRepository {
	return &promptRepository{db: db}
}

func (repo *promptRepository) SavePrompt(ctx context.Context, p *domain.Prompt) error {
	args := []any{p.MessageID, p.ChatID, p.Text, p.FromUser}
	q := `insert into prompts (message_id, chat_id, text, created_by) values ($1, $2, $3, $4)`

	if _, err := repo.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("saving prompt: %w", err)
	}

	return nil
}

func (repo *promptRepository) FetchPrompt(ctx context.Context, chatID int64, messageID int) (*domain.Prompt, error) {
	q := `
		select message_id,
		    chat_id,   
			text,
			created_by
		from prompts
		where chat_id = $1
		and message_id = $2
	`

	p := domain.Prompt{}
	if err := repo.db.QueryRowContext(ctx, q, chatID, messageID).Scan(
		&p.MessageID,
		&p.ChatID,
		&p.Text,
		&p.FromUser,
	); err != nil {
		return nil, fmt.Errorf("scanning prompt row: %w", err)
	}

	return &p, nil
}
