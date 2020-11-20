package rater

import (
	"context"
	"fmt"

	"github.com/Zetkolink/oracle/models/evaluations"
	"github.com/Zetkolink/oracle/models/forRate"
	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
)

type Rater struct {
	models ModelsSet
}

type Config struct {
	Models ModelsSet
}

type ModelsSet struct {
	Goals       *goals.Model
	UserGoals   *userGoals.Model
	Evaluations *evaluations.Model
	ForRate     *forRate.Model
}

func NewRater(config Config) *Rater {
	return &Rater{models: config.Models}
}

func (r *Rater) GetToRate(ctx context.Context, user *users.User) (*userGoals.UserGoal, string, error) {
	fRate, err := r.models.ForRate.Get(ctx, user.ID)

	if err != nil {
		return nil, "", err
	}

	if fRate == nil {
		return nil, "", nil
	}

	uGoal, err := r.models.UserGoals.Get(ctx, fRate.UserGoalID)

	if err != nil {
		return nil, "", err
	}

	goal, err := r.models.Goals.Get(ctx, uGoal.GoalID)

	if err != nil {
		return nil, "", err
	}

	uDate, err := user.Date(uGoal.From)

	if err != nil {
		return nil, "", err
	}

	message := fmt.Sprintf("Пользователь\n 🙍‍♂ - %d\nДата\n ⏱ - %s\nЗадача\n 💡 - %s",
		uGoal.UserID+user.ID, uDate.Format("January 2"), goal.Description)

	return uGoal, message, nil
}

func (r *Rater) Rate(ctx context.Context, user int64, uGoal int64, eval bool) error {
	ev := &evaluations.Evaluation{
		UserID:     user,
		UserGoalID: uGoal,
		Evaluation: eval,
	}

	err := r.models.Evaluations.Create(ctx, ev)

	if err != nil {
		return err
	}

	return nil
}
