package userGoals

import (
	"context"
	"database/sql"
	"time"
)

const (
	StatusSoon       = "soon"
	StatusInProgress = "inProgress"
	StatusComplete   = "complete"
	StatusFailed     = "failed"

	PhasePlanning = "planning"
	PhaseActive   = "active"
	PhaseFinished = "finished"
)

// Model type represent model.
type Model struct {
	db *sql.DB
}

// ModelConfig type represent model config.
type ModelConfig struct {
	Db *sql.DB
}

// ModelConfig type represent user goal.
type UserGoal struct {
	ID     int64     `json:"id"`
	UserID int64     `json:"user_id"`
	GoalID int64     `json:"goal_id"`
	Type   int64     `json:"type"`
	Phase  string    `json:"phase"`
	Status string    `json:"status"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db: config.Db,
	}

	return m, nil
}

// Create create new user goal.
func (m *Model) Create(ctx context.Context, uGoal *UserGoal) error {
	_, err := m.db.ExecContext(ctx, `INSERT INTO user_goals
									( "user_id", "goal_id", "type",
									 "phase","status", "from", "to")
								VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uGoal.UserID, uGoal.GoalID, uGoal.Type,
		uGoal.Phase, uGoal.Status, uGoal.From, uGoal.To)

	if err != nil {
		return err
	}

	return nil
}

// Get get user goal by ID.
func (m *Model) Get(ctx context.Context, id int64) (*UserGoal, error) {
	var uGoal UserGoal

	err := m.db.QueryRowContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									 "phase","status", "from", "to"
									     FROM user_goals
								WHERE id = $1`, id).
		Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type, &uGoal.Phase,
			&uGoal.Status, &uGoal.From, &uGoal.To)

	if err != nil {
		return nil, err
	}

	return &uGoal, nil
}

// Delete delete user goal by ID.
func (m *Model) Delete(ctx context.Context, id int64) error {
	_, err := m.db.ExecContext(ctx, `DELETE  FROM user_goals
								WHERE id = $1`, id)

	if err != nil {
		return err
	}

	return nil
}

// UpdateGoal update user goal id.
func (m *Model) UpdateGoal(ctx context.Context, id int64, goalID int64) error {
	_, err := m.db.ExecContext(ctx, `UPDATE user_goals
								SET goal_id = $2
								WHERE id = $1`, id, goalID)

	if err != nil {
		return err
	}

	return nil
}

// UpdatePhase update user goal phase.
func (m *Model) UpdatePhase(ctx context.Context, id int64, phase string) error {
	_, err := m.db.ExecContext(ctx, `UPDATE user_goals
								SET phase = $2
								WHERE id = $1`, id, phase)

	if err != nil {
		return err
	}

	return nil
}

// UpdateStatus update user goal status.
func (m *Model) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := m.db.ExecContext(ctx, `UPDATE user_goals
								SET status = $2
								WHERE id = $1`, id, status)

	if err != nil {
		return err
	}

	return nil
}

// List get user goals.
func (m *Model) List(ctx context.Context) ([]*UserGoal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									"phase","status", "from", "to"
									FROM user_goals`)

	if err != nil {
		return nil, err
	}

	var uGoals []*UserGoal

	for rows.Next() {
		var uGoal UserGoal

		err = rows.Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type,
			&uGoal.Phase, &uGoal.Status, &uGoal.From, &uGoal.To)

		if err != nil {
			return nil, err
		}

		uGoals = append(uGoals, &uGoal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uGoals, nil
}

// List get user goals by date.
func (m *Model) ListByDate(ctx context.Context, date time.Time) ([]*UserGoal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									"phase","status", "from", "to"
									FROM user_goals
									WHERE $1 > "from"
									AND $1 < "to"`, date)

	if err != nil {
		return nil, err
	}

	var uGoals []*UserGoal

	for rows.Next() {
		var uGoal UserGoal

		err = rows.Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type,
			&uGoal.Phase, &uGoal.Status, &uGoal.From, &uGoal.To)

		if err != nil {
			return nil, err
		}

		uGoals = append(uGoals, &uGoal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uGoals, nil
}

// List get user goals by user.
func (m *Model) ListByUser(ctx context.Context, userID int64) ([]*UserGoal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									"phase","status", "from", "to"
									FROM user_goals
									WHERE $1 = "user_id"`, userID)

	if err != nil {
		return nil, err
	}

	var uGoals []*UserGoal

	for rows.Next() {
		var uGoal UserGoal

		err = rows.Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type,
			&uGoal.Phase, &uGoal.Status, &uGoal.From, &uGoal.To)

		if err != nil {
			return nil, err
		}

		uGoals = append(uGoals, &uGoal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uGoals, nil
}

// ListByUserAndDate get user goals by user and date.
func (m *Model) ListByUserAndDate(ctx context.Context, userID int64, date *time.Time) ([]*UserGoal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									"phase","status", "from", "to"
									FROM user_goals
									WHERE $1 = "user_id"
									AND $2 >= "from" AND $2 <= "to"
									`, userID, date)

	if err != nil {
		return nil, err
	}

	var uGoals []*UserGoal

	for rows.Next() {
		var uGoal UserGoal

		err = rows.Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type,
			&uGoal.Phase, &uGoal.Status, &uGoal.From, &uGoal.To)

		if err != nil {
			return nil, err
		}

		uGoals = append(uGoals, &uGoal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uGoals, nil
}

// ListByParams get user goals by user and date.
func (m *Model) ListByParams(ctx context.Context, userID int64, date *time.Time, gType int64) ([]*UserGoal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									"phase","status", "from", "to"
									FROM user_goals
									WHERE $1 = "user_id"
									AND $2 >= "from" AND $2 <= "to"
									AND $3 = "type"
									`, userID, date, gType)

	if err != nil {
		return nil, err
	}

	var uGoals []*UserGoal

	for rows.Next() {
		var uGoal UserGoal

		err = rows.Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type,
			&uGoal.Phase, &uGoal.Status, &uGoal.From, &uGoal.To)

		if err != nil {
			return nil, err
		}

		uGoals = append(uGoals, &uGoal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uGoals, nil
}

// List get user goals by phase.
func (m *Model) ListByPhase(ctx context.Context, phase string) ([]*UserGoal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "user_id", "goal_id", "type",
									"phase","status", "from", "to"
									FROM user_goals
									WHERE $1 = "phase"`, phase)

	if err != nil {
		return nil, err
	}

	var uGoals []*UserGoal

	for rows.Next() {
		var uGoal UserGoal

		err = rows.Scan(&uGoal.ID, &uGoal.UserID, &uGoal.GoalID, &uGoal.Type,
			&uGoal.Phase, &uGoal.Status, &uGoal.From, &uGoal.To)

		if err != nil {
			return nil, err
		}

		uGoals = append(uGoals, &uGoal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uGoals, nil
}
