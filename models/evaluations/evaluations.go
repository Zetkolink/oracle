package evaluations

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

// ModelConfig type represent evaluation.
type Evaluation struct {
	ID         int64 `json:"id"`
	UserID     int64 `json:"user_id"`
	UserGoalID int64 `json:"user_goal_id"`
	Evaluation bool  `json:"evaluation"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db: config.Db,
	}

	return m, nil
}

// Create create new evaluation.
func (m *Model) Create(ctx context.Context, eval *Evaluation) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO evaluations
    			("user_goal_id", "user_id" ,"evaluation") VALUES ($1, $2, $3)`,
		eval.UserGoalID, eval.UserID, eval.Evaluation)

	if err != nil {
		return err
	}

	return nil
}

// Get get evaluation result by user goal.
func (m *Model) GetResult(ctx context.Context, uGoalID int64) (bool, error) {
	var value int64

	err := m.db.QueryRowContext(ctx, `SELECT 
        sum(CASE WHEN evaluation THEN 1 ELSE -1 END)  
		from evaluations
		WHERE user_goal_id = $1`, uGoalID).
		Scan(&value)

	if err != nil {
		return false, err
	}

	if value >= 0 {
		return true, nil
	}

	return false, nil
}

// List get evaluations list.
func (m *Model) List(ctx context.Context, user int64) ([]*Evaluation, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_goal_id", "user_id", "evaluation"
									FROM evaluations
									WHERE "user_id" = $1`, user)

	if err != nil {
		return nil, err
	}

	var evals []*Evaluation

	for rows.Next() {
		var eval Evaluation

		err = rows.Scan(&eval.ID, &eval.UserGoalID, &eval.UserID,
			&eval.Evaluation)

		if err != nil {
			return nil, err
		}

		evals = append(evals, &eval)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return evals, nil
}
