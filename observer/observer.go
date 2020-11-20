package observer

import (
	"context"
	"log"
	"time"

	"github.com/Zetkolink/oracle/models/userGoals"
)

type Observer struct {
	models ModelsSet
}

type Config struct {
	Models ModelsSet
}

type ModelsSet struct {
	UserGoals *userGoals.Model
}

func NewObserver(config Config) *Observer {
	return &Observer{models: config.Models}
}

func (o *Observer) Run() {
	go func() {
		err := o.UpdateActive(context.Background())

		if err != nil {
			log.Println(err)
		}

		err = o.UpdatePlanning(context.Background())

		if err != nil {
			log.Println(err)
		}

		time.Sleep(1 * time.Hour)
	}()
}

func (o *Observer) UpdatePlanning(ctx context.Context) error {
	uGoals, err := o.models.UserGoals.ListByPhase(ctx, userGoals.PhasePlanning)

	if err != nil {
		return err
	}

	for _, uGoal := range uGoals {
		if time.Now().UTC().After(uGoal.From) {
			err := o.models.UserGoals.UpdatePhase(ctx, uGoal.ID, userGoals.PhaseActive)

			if err != nil {
				return err
			}

			err = o.models.UserGoals.UpdateStatus(ctx, uGoal.ID,
				userGoals.StatusInProgress)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *Observer) UpdateActive(ctx context.Context) error {
	uGoals, err := o.models.UserGoals.ListByPhase(ctx, userGoals.PhaseActive)

	if err != nil {
		return err
	}

	for _, uGoal := range uGoals {
		if time.Now().UTC().After(uGoal.To) {
			err := o.models.UserGoals.UpdatePhase(ctx, uGoal.ID,
				userGoals.PhaseFinished)

			if err != nil {
				return err
			}

			if uGoal.Status == userGoals.StatusInProgress {
				err := o.models.UserGoals.UpdateStatus(ctx, uGoal.ID,
					userGoals.StatusFailed)

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
