package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dskvich/chatgpt-telegram-bot/pkg/domain"
)

type chatStyleRepository struct {
	db *sql.DB
}

func NewChatStyleRepository(db *sql.DB) *chatStyleRepository {
	return &chatStyleRepository{db: db}
}

func (r *chatStyleRepository) NewStyleFromActive(ctx context.Context, chatID int64, name, createdBy string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// Get the current active style description
	var currentDescription string
	err = tx.QueryRowContext(ctx, `
        SELECT description
        FROM chat_styles
        WHERE chat_id = $1 AND is_active = TRUE
    `, chatID).Scan(&currentDescription)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("no active style found")
		}
		return err
	}

	// Create a new style based on the current active style
	_, err = tx.ExecContext(ctx, `
        INSERT INTO chat_styles (chat_id, name, is_active, description, created_by)
        VALUES ($1, $2, FALSE, $3, $4)
    `, chatID, name, currentDescription, createdBy)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

func (r *chatStyleRepository) GetActiveStyle(ctx context.Context, chatID int64) (*domain.ChatStyle, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT chat_id, name, is_active, description, created_by
        FROM chat_styles
        WHERE chat_id = $1 AND is_active = TRUE
    `, chatID)

	var chatStyle domain.ChatStyle
	if err := row.Scan(
		&chatStyle.ChatID,
		&chatStyle.Name,
		&chatStyle.IsActive,
		&chatStyle.Description,
		&chatStyle.CreatedBy,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &chatStyle, nil
}

func (r *chatStyleRepository) Activate(ctx context.Context, chatID int64, name string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// Ensure the style exists (case-insensitive)
	var exists bool
	err = tx.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM chat_styles
            WHERE chat_id = $1 AND LOWER(name) = LOWER($2)
        )
    `, chatID, name).Scan(&exists)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	if !exists {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return errors.New("style not found")
	}

	// Deactivate all other styles for the chat
	_, err = tx.ExecContext(ctx, `
        UPDATE chat_styles
        SET is_active = FALSE
        WHERE chat_id = $1
    `, chatID)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	// Activate the specified style (case-insensitive)
	result, err := tx.ExecContext(ctx, `
        UPDATE chat_styles
        SET is_active = TRUE
        WHERE chat_id = $1 AND LOWER(name) = LOWER($2)
    `, chatID, name)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	if affectedRows == 0 {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("rollback failed: %v", rollbackErr)
		}
		return errors.New("style activation failed")
	}

	return tx.Commit()
}

func (r *chatStyleRepository) UpdateActiveStyle(ctx context.Context, chatID int64, newInstruction string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// Fetch the current description of the active style
	var currentDescription string
	err = tx.QueryRowContext(ctx, `
        SELECT description
        FROM chat_styles
        WHERE chat_id = $1 AND is_active = TRUE
    `, chatID).Scan(&currentDescription)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No active style found, create a new one
			_, err = tx.ExecContext(ctx, `
                INSERT INTO chat_styles (chat_id, name, is_active, description, created_by)
                VALUES ($1, 'default', TRUE, $2, '')
            `, chatID, newInstruction)
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
				}
				return err
			}
			return tx.Commit()
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	updatedDescription := currentDescription + " " + newInstruction

	// Update the description of the currently active style
	_, err = tx.ExecContext(ctx, `
        UPDATE chat_styles
        SET description = $1
        WHERE chat_id = $2 AND is_active = TRUE
    `, updatedDescription, chatID)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("query failed: %v, and rollback also failed: %v", err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

func (r *chatStyleRepository) GetAllStyles(ctx context.Context, chatID int64) ([]domain.ChatStyle, error) {
	query := `
        SELECT name, description, is_active
        FROM chat_styles
        WHERE chat_id = $1
        ORDER BY created_at
    `

	rows, err := r.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var styles []domain.ChatStyle
	for rows.Next() {
		var style domain.ChatStyle
		if err := rows.Scan(&style.Name, &style.Description, &style.IsActive); err != nil {
			return nil, err
		}
		styles = append(styles, style)
	}

	return styles, nil
}
