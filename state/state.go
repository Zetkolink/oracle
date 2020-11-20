package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type State struct {
	Key    string      `json:"key"`
	PeerID int64       `json:"peer_id"`
	Params interface{} `json:"params"`
	redis  *redis.Client
}

func NewState(ctx context.Context, peerID int64, prefix string, r *redis.Client) (*State, error) {
	key := fmt.Sprintf("%s_%d", prefix, peerID)
	stateRaw, err := r.Get(ctx, key).Result()

	if err != nil && err != redis.Nil {
		return nil, err
	}

	if err == redis.Nil {
		state := &State{
			Key:    key,
			PeerID: peerID,
			redis:  r,
		}

		jsState, err := json.Marshal(state)

		err = r.Set(ctx, key, jsState, 0).Err()

		if err != nil {
			return nil, err
		}

		return state, nil
	}

	var state State

	err = json.Unmarshal([]byte(stateRaw), &state)

	if err != nil {
		return nil, err
	}

	state.redis = r

	return &state, nil
}

func (s *State) SetParams(ctx context.Context, params interface{}) error {
	s.Params = params
	jsState, err := json.Marshal(s)

	if err != nil {
		return err
	}

	err = s.redis.Set(ctx, s.Key, jsState, 0).Err()

	if err != nil {
		return err
	}

	return nil
}

func (s *State) Clear(ctx context.Context) {
	s.redis.Del(ctx, s.Key)
}
