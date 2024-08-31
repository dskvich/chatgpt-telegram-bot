package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type promptRepository struct {
	db *sql.DB
}

func NewPromptRepository(db *sql.DB) *promptRepository {
	return &promptRepository{db: db}
}

func (repo *promptRepository) SavePrompt(ctx context.Context, p *domain.Prompt) error {
	columns := []string{"message_id", "chat_id", "text", "created_by"}
	args := []any{p.MessageID, p.ChatID, p.Text, p.FromUser}

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	q := `insert into prompts (` + strings.Join(columns, ", ") + `) values (` + strings.Join(placeholders, ",") + `)`

	if _, err := repo.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("saving prompt: %v", err)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scanning prompt row: %v", err)
	}

	return &p, nil
}
