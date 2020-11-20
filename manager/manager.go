package manager

import (
	"context"
	"errors"
	"time"

	"github.com/Zetkolink/oracle/models/goalTypes"
	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
)

var (
	ErrConflict = errors.New("goal conflict")
)

type Manager struct {
	models ModelsSet
}

type Config struct {
	Models ModelsSet
}

type ModelsSet struct {
	Users     *users.Model
	Goals     *goals.Model
	GoalTypes *goalTypes.Model
	UserGoals *userGoals.Model
}

func NewManager(config Config) *Manager {
	return &Manager{models: config.Models}
}

func (m *Manager) AssignGoal(ctx context.Context, user *users.User,
	goal *goals.Goal, date time.Time) (*userGoals.UserGoal, error) {

	ok, err := m.models.Goals.Isset(ctx, goal.ID)

	if err != nil {
		return nil, err
	}

	if !ok {
		goal.ID, err = m.models.Goals.Create(ctx, goal)

		if err != nil {
			return nil, err
		}
	}

	uDate, err := user.Date(date)

	if err != nil {
		return nil, err
	}

	uGoal, err := m.GetByType(ctx, user, *uDate, goal.Type)

	if err != nil {
		return nil, err
	}

	if uGoal != nil {
		err := m.models.UserGoals.UpdateGoal(ctx, uGoal.ID, goal.ID)

		if err != nil {
			return nil, err
		}

		return uGoal, nil
	}

	from, err := user.StartDate(date)

	if err != nil {
		return nil, err
	}

	to, err := user.EndDate(date)

	if err != nil {
		return nil, err
	}

	uGoal = &userGoals.UserGoal{
		UserID: user.ID,
		GoalID: goal.ID,
		Phase:  userGoals.PhasePlanning,
		Status: userGoals.StatusSoon,
		From:   from.UTC(),
		To:     to.UTC(),
		Type:   goal.Type,
	}

	err = m.models.UserGoals.Create(ctx, uGoal)

	if err != nil {
		return nil, err
	}

	return uGoal, nil
}

func (m *Manager) SetPhase(ctx context.Context, uGoalID int64, phase string) error {
	err := m.models.UserGoals.UpdatePhase(ctx, uGoalID, phase)

	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) SetStatus(ctx context.Context, user *users.User, date time.Time, gType int64) error {
	uGoals, err := m.UserGoals(ctx, user, date)

	if err != nil {
		return err
	}

	for _, uGoal := range uGoals {
		if uGoal.Type == gType {
			status := userGoals.StatusComplete

			if uGoal.Status == userGoals.StatusComplete {
				status = userGoals.StatusInProgress
			}

			err := m.models.UserGoals.UpdateStatus(ctx, uGoal.ID, status)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Manager) RejectGoal(ctx context.Context, uGoalID int64) error {
	err := m.models.UserGoals.Delete(ctx, uGoalID)

	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) CheckDate(ctx context.Context, user *users.User,
	date time.Time) (bool, error) {
	types, err := m.models.GoalTypes.List(ctx)

	if err != nil {
		return false, err
	}

	typeIDs := make(map[int64]bool)

	for _, gType := range types {
		typeIDs[gType.ID] = false
	}

	uDate, err := user.Date(date)

	if err != nil {
		return false, err
	}

	gls, err := m.models.UserGoals.ListByUserAndDate(ctx, user.ID,
		uDate)

	if err != nil {
		return false, err
	}

	for _, goal := range gls {
		typeIDs[goal.Type] = true
	}

	for _, ok := range typeIDs {
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func (m *Manager) GetByType(ctx context.Context, user *users.User,
	date time.Time, gType int64) (*userGoals.UserGoal, error) {
	uDate, err := user.Date(date)

	if err != nil {
		return nil, err
	}

	gls, err := m.models.UserGoals.ListByUserAndDate(ctx, user.ID,
		uDate)

	if err != nil {
		return nil, err
	}

	for _, goal := range gls {
		if goal.Type == gType {
			return goal, nil
		}
	}

	return nil, nil
}

func (m *Manager) UserGoals(ctx context.Context, user *users.User,
	date time.Time) ([]*userGoals.UserGoal, error) {
	uDate, err := user.Date(date)

	if err != nil {
		return nil, err
	}

	gls, err := m.models.UserGoals.ListByUserAndDate(ctx, user.ID,
		uDate)

	if err != nil {
		return nil, err
	}

	return gls, nil
}
