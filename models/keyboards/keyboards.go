package keyboards

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

// Keyboard type represent keyboard.
type Keyboard struct {
	ID       int64  `json:"id"`
	Service  string `json:"service"`
	Name     string `json:"name"`
	Keyboard string `json:"keyboard"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db: config.Db,
	}

	return m, nil
}

// Get get keyboard by ID.
func (m *Model) GetKeyboard(ctx context.Context, service string, keyboard string) (string, error) {
	var kb string

	err := m.db.QueryRowContext(ctx, `SELECT  
									 "keyboard"
									 FROM keyboards
								WHERE service = $1
								AND name = $2`, service, keyboard).
		Scan(&kb)

	if err != nil {
		return "", err
	}

	return kb, nil
}
