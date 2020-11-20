package whiteList

import (
	"context"
	"database/sql"
)

// Model type represent model.
type Model struct {
	db *sql.DB
}

// ModelConfig type represent model config.
type ModelConfig struct {
	Db *sql.DB
}

// ModelConfig type represent white list item.
type Item struct {
	UserID int64 `json:"user_id"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db: config.Db,
	}

	return m, nil
}

// Create create new item.
func (m *Model) Create(ctx context.Context, item *Item) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO white_list
    			("user_id") VALUES ($1)`,
		item.UserID)

	if err != nil {
		return err
	}

	return nil
}

// Check check user in white list.
func (m *Model) Check(ctx context.Context, userID int64) (bool, error) {
	err := m.db.QueryRowContext(ctx, `SELECT "user_id" 
		FROM white_list
		WHERE "user_id" = $1`, userID).
		Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
