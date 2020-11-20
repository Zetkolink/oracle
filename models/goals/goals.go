package goals

import (
	"context"
	"database/sql"
)

var (
	Awaking6  = 1
	Awaking8  = 2
	Awaking10 = 3
)

// Model type represent model.
type Model struct {
	db *sql.DB
}

// ModelConfig type represent model config.
type ModelConfig struct {
	Db *sql.DB
}

// ModelConfig type represent goal.
type Goal struct {
	ID          int64  `json:"id"`
	Type        int64  `json:"type"`
	Description string `json:"description"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db: config.Db,
	}

	return m, nil
}

// Create create new goal.
func (m *Model) Create(ctx context.Context, goal *Goal) (int64, error) {
	var id int64

	err := m.db.QueryRowContext(ctx,
		`INSERT INTO goals ("type","description") 
				VALUES ($1, $2)
				RETURNING "id"`,
		goal.Type, goal.Description).Scan(&id)

	if err != nil {
		return id, err
	}

	return id, nil
}

// Get get goal by ID.
func (m *Model) Get(ctx context.Context, id int64) (*Goal, error) {
	var goal Goal

	err := m.db.QueryRowContext(ctx, `SELECT  
									"id", "type","description"
									 FROM goals
								WHERE id = $1`, id).
		Scan(&goal.ID, &goal.Type, &goal.Description)

	if err != nil {
		return nil, err
	}

	return &goal, nil
}

// Isset check goal is set.
func (m *Model) Isset(ctx context.Context, id int64) (bool, error) {
	err := m.db.QueryRowContext(ctx, `SELECT "id" FROM goals
								WHERE id = $1`, id).Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// List get goals.
func (m *Model) List(ctx context.Context, gType int64) ([]*Goal, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "type","description"
									FROM goals
									WHERE "type" = $1`, gType)

	if err != nil {
		return nil, err
	}

	var goals []*Goal

	for rows.Next() {
		var goal Goal

		err = rows.Scan(&goal.ID, &goal.Type, &goal.Description)

		if err != nil {
			return nil, err
		}

		goals = append(goals, &goal)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return goals, nil
}

func (g *Goal) GetItem() interface{} {
	return g.ID
}

func (g *Goal) GetLabel() string {
	return g.Description
}
