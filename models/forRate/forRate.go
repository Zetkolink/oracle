package forRate

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

// ForRate type represent forRate.
type ForRate struct {
	UserID     int64 `json:"user_id"`
	UserGoalID int64 `json:"user_goal_id"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db: config.Db,
	}

	return m, nil
}

// Get get for rate by user ID.
func (m *Model) Get(ctx context.Context, userID int64) (*ForRate, error) {
	var forRate ForRate

	err := m.db.QueryRowContext(ctx, `SELECT  
									"user_id", "user_goal_id"
									 FROM for_rate
								WHERE "user_id" = $1`, userID).
		Scan(&forRate.UserID, &forRate.UserGoalID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &forRate, nil
}
