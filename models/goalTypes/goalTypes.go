package goalTypes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

const Awaking = 1

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

// GoalType type represent goal type.
type GoalType struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Points    int64  `json:"points"`
	Evaluated bool   `json:"evaluated"`
	FromList  bool   `json:"from_list"`
}

// NewModel create new Model.
func NewModel(config ModelConfig) (*Model, error) {
	return &Model{
		db:    config.Db,
		cache: config.Cache,
	}, nil
}

// Get get goal type by id.
func (m *Model) Get(ctx context.Context, id int64) (*GoalType, error) {
	var gt GoalType

	cachedGt, err := m.getCache(ctx, id)

	if err != nil && err != redis.Nil {
		log.Println(err)
	}

	if cachedGt != nil {
		return cachedGt, nil
	}

	err = m.db.QueryRowContext(ctx, `SELECT  
									"id", "name","points", 
       								"evaluated", "from_list"
									     FROM goal_types
								WHERE "id" = $1`,
		id,
	).Scan(&gt.ID, &gt.Name, &gt.Points, &gt.Evaluated, &gt.FromList)

	if err != nil {
		return nil, err
	}

	return &gt, nil
}

// List get goal types.
func (m *Model) List(ctx context.Context) ([]*GoalType, error) {
	gtsCached, err := m.listCache(ctx)

	if err != nil && err != redis.Nil {
		log.Println(err)
	}

	if gtsCached != nil {
		return gtsCached, nil
	}

	rows, err := m.db.QueryContext(ctx, `SELECT  
									"id", "name","points", 
       								"evaluated", "from_list"
									FROM goal_types
									ORDER BY "id"`)

	if err != nil {
		return nil, err
	}

	var gts []*GoalType

	for rows.Next() {
		var gt GoalType

		err = rows.Scan(&gt.ID, &gt.Name, &gt.Points, &gt.Evaluated,
			&gt.FromList)

		if err != nil {
			return nil, err
		}

		gts = append(gts, &gt)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	err = m.setListCache(ctx, gts)

	if err != nil {
		log.Println(err)
	}

	return gts, nil
}

func (m *Model) listCache(ctx context.Context) ([]*GoalType, error) {
	var gTypes []*GoalType

	raw, err := m.cache.Get(ctx, "goal_type_list").Result()

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(raw), &gTypes)

	if err != nil {
		return nil, err
	}

	return gTypes, nil
}

func (m *Model) setListCache(ctx context.Context, gTypes []*GoalType) error {
	rawBytes, err := json.Marshal(gTypes)

	if err != nil {
		return err
	}

	err = m.cache.Set(ctx, "goal_type_list", rawBytes, 0).Err()

	if err != nil {
		return err
	}

	return nil
}

func (m *Model) getCache(ctx context.Context, id int64) (*GoalType, error) {
	var gType GoalType

	raw, err := m.cache.Get(ctx, m.key(id)).Result()

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(raw), &gType)

	if err != nil {
		return nil, err
	}

	return &gType, nil
}

func (m *Model) setCache(ctx context.Context, gType *GoalType) error {
	rawBytes, err := json.Marshal(gType)

	if err != nil {
		return err
	}

	err = m.cache.Set(ctx, m.key(gType.ID), rawBytes, 0).Err()

	if err != nil {
		return err
	}

	return nil
}

func (m *Model) key(id int64) string {
	return fmt.Sprintf("goal_type_%d", id)
}
