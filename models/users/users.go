package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	"github.com/lib/pq"
)

var (
	// ErrExists user exists.
	ErrExists = errors.New("user exists")
)

// Model type represent model.
type Model struct {
	db    *sql.DB
	cache *redis.Client
}

// ModelConfig type represent model config.
type ModelConfig struct {
	Db    *sql.DB
	Cache *redis.Client
}

// ModelConfig type represent user.
type User struct {
	ID        int64      `json:"id"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	City      string     `json:"city"`
	Timezone  string     `json:"timezone"`
	Active    bool       `json:"status"`
	State     string     `json:"state"`
	CreatedAt *time.Time `json:"created_at"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	m := &Model{
		db:    config.Db,
		cache: config.Cache,
	}

	return m, nil
}

// Create create new user.
func (m *Model) Create(ctx context.Context, user *User) error {
	_, err := m.db.ExecContext(ctx, `INSERT INTO users
									( "id", "first_name","last_name", 
									 "timezone", "city", "state")
								VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.FirstName, user.LastName,
		user.Timezone, user.City, user.State)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			if pgErr.Code == "23505" {
				return ErrExists
			}
		}

		return err
	}

	return nil
}

// List get user list.
func (m *Model) List(ctx context.Context) ([]*User, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "first_name","last_name", 
       								"active", "timezone", "created_at",
									"state", "city"
									FROM users
									ORDER BY "id"`)

	if err != nil {
		return nil, err
	}

	var users []*User

	for rows.Next() {
		var user User

		err = rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Active,
			&user.Timezone, &user.CreatedAt, &user.State, &user.City)

		if err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return users, nil
}

// Get get user by ID.
func (m *Model) Get(ctx context.Context, id int64) (*User, error) {
	var user User

	cachedUser, err := m.getCache(ctx, id)

	if err != nil && err != redis.Nil {
		log.Println(err)
	}

	if cachedUser != nil {
		return cachedUser, nil
	}

	err = m.db.QueryRowContext(ctx, `SELECT  
									"id", "first_name","last_name", 
       								"active", "timezone", "created_at",
									"state", "city"
									     FROM users
								WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Active,
		&user.Timezone, &user.CreatedAt, &user.State, &user.City)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	err = m.setCache(ctx, &user)

	if err != nil {
		log.Println(err)
	}

	return &user, nil
}

func (m *Model) getCache(ctx context.Context, id int64) (*User, error) {
	var user User

	userRaw, err := m.cache.Get(ctx, m.key(id)).Result()

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(userRaw), &user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (m *Model) setCache(ctx context.Context, user *User) error {
	userRawBytes, err := json.Marshal(user)

	if err != nil {
		return err
	}

	err = m.cache.Set(ctx, m.key(user.ID), userRawBytes, 0).Err()

	if err != nil {
		return err
	}

	return nil
}

// UpdateState update user state.
func (m *Model) UpdateState(ctx context.Context, userID int64, state string) error {
	_, err := m.db.ExecContext(ctx, `UPDATE users SET
									state = $2 WHERE id = $1`,
		userID, state)

	if err != nil {
		return err
	}

	err = m.cache.Del(ctx, m.key(userID)).Err()

	if err != nil {
		log.Println(err)
	}

	return nil
}

func (m *Model) key(id int64) string {
	return fmt.Sprintf("user_%d", id)
}

// GetTime get user time.
func (u *User) GetTime() (*time.Time, error) {
	location, err := time.LoadLocation(u.Timezone)

	if err != nil {
		return nil, err
	}

	userTime := time.Now().In(location)

	return &userTime, nil
}

// StartDate get start date in user location.
func (u *User) Date(t time.Time) (*time.Time, error) {
	location, err := time.LoadLocation(u.Timezone)

	if err != nil {
		return nil, err
	}

	date := t.In(location)

	return &date, nil
}

// StartDate get start date in user location.
func (u *User) StartDate(t time.Time) (*time.Time, error) {
	location, err := time.LoadLocation(u.Timezone)

	if err != nil {
		return nil, err
	}

	year, month, day := t.In(location).Date()
	start := time.Date(year, month, day,
		6, 0, 0, 0, t.Location())

	return &start, nil
}

// EndDate get end date in user location.
func (u *User) EndDate(t time.Time) (*time.Time, error) {
	start, err := u.StartDate(t)

	if err != nil {
		return nil, err
	}

	end := start.Add(24*time.Hour - time.Second)

	return &end, nil
}
