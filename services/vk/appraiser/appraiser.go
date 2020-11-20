package appraiser

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/Zetkolink/oracle/notificator"
	"github.com/Zetkolink/oracle/rater"
	"github.com/Zetkolink/oracle/services"
	"github.com/Zetkolink/oracle/services/vk/keyboard"
	"github.com/go-redis/redis/v8"
	"github.com/go-vk-api/vk"
)

var (
	menuBtn = &keyboard.Button{
		Color: "secondary",
		Action: keyboard.Action{
			Label: "Меню",
			Type:  "text",
			Payload: keyboard.Payload{
				Command: "menu",
			},
		},
	}
)

type Appraiser struct {
	vkClient    *vk.Client
	rater       *rater.Rater
	redisClient *redis.Client
	models      ModelsSet
	notificator *notificator.Notificator
}

type Config struct {
	VKClient    *vk.Client
	Rater       *rater.Rater
	Models      ModelsSet
	Notificator *notificator.Notificator
}

type ModelsSet struct {
	Users     *users.Model
	UserGoals *userGoals.Model
	Goals     *goals.Model
}

func NewAppraiser(config Config) *Appraiser {
	return &Appraiser{
		vkClient:    config.VKClient,
		rater:       config.Rater,
		models:      config.Models,
		notificator: config.Notificator,
	}
}

func (r *Appraiser) Handle(ctx context.Context, message services.Message) (string, error) {
	payload, err := message.GetPayload()

	if err != nil {
		return "", err
	}

	if payload == nil {
		err := r.SendMain(ctx, message.GetUser())

		if err != nil {
			return "", err
		}

		return "", nil
	}

	switch payload.GetCommand() {
	case "menu":
		err = r.models.Users.UpdateState(ctx, message.GetPeer(), "menu")

		if err != nil {
			return "", err
		}

		return "menu", nil
	case "approve", "disapprove":
		uGoalParam, ok := payload.GetParam("user_goal").(float64)

		if !ok {
			return "", errors.New("user goal not found")
		}

		uGoalID := int64(uGoalParam)

		var eval bool

		if payload.GetCommand() == "approve" {
			eval = true
		}

		err = r.rater.Rate(ctx, message.GetPeer(), uGoalID, eval)

		if err != nil {
			return "", err
		}

		if !eval {
			go func() {
				err := r.Notify(ctx, uGoalID)

				if err != nil {
					log.Println(err)
				}
			}()
		}

		err := r.SendMain(ctx, message.GetUser())

		if err != nil {
			return "", err
		}

		return "", nil
	default:
		err := r.SendMain(ctx, message.GetUser())

		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func (r *Appraiser) SendMain(ctx context.Context, user *users.User) error {
	kb := keyboard.NewKeyboard(keyboard.Config{
		OneTime: true,
		Inline:  false,
		Width:   2,
		Height:  1,
	})

	kb.SetFooter(menuBtn)

	uGoal, goal, err := r.rater.GetToRate(ctx, user)

	if err != nil {
		return err
	}

	if uGoal == nil {
		kbStr, err := kb.Marshal()

		if err != nil {
			return err
		}

		err = r.vkClient.CallMethod("messages.send", vk.RequestParams{
			"peer_id":   user.ID,
			"message":   "На данный момент вы оценили все",
			"random_id": 0,
			"keyboard":  kbStr,
		}, nil)

		if err != nil {
			return err
		}

		return nil
	}

	approveBtn := &keyboard.Button{
		Color: "positive",
		Action: keyboard.Action{
			Label: "👍🏻",
			Type:  "text",
			Payload: keyboard.Payload{
				Command: "approve",
				Params: map[string]interface{}{
					"user_goal": uGoal.ID,
				},
			},
		},
	}

	kb.SetButton(0, 0, approveBtn)

	disapproveBtn := &keyboard.Button{
		Color: "negative",
		Action: keyboard.Action{
			Label: "👎🏻",
			Type:  "text",
			Payload: keyboard.Payload{
				Command: "disapprove",
				Params: map[string]interface{}{
					"user_goal": uGoal.ID,
				},
			},
		},
	}

	kb.SetButton(0, 1, disapproveBtn)

	kbStr, err := kb.Marshal()

	if err != nil {
		return err
	}

	err = r.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   goal,
		"random_id": 0,
		"keyboard":  kbStr,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (r *Appraiser) Notify(ctx context.Context, uGoalID int64) error {
	userGoal, err := r.models.UserGoals.Get(ctx, uGoalID)

	if err != nil {
		return err
	}

	goal, err := r.models.Goals.Get(ctx, userGoal.GoalID)

	if err != nil {
		return err
	}

	user, err := r.models.Users.Get(ctx, userGoal.UserID)

	if err != nil {
		return err
	}

	uDate, err := user.Date(userGoal.From)

	if err != nil {
		return err
	}

	err = r.notificator.Send(user, "disapprove",
		fmt.Sprintf("Ваша задача была помечена как невалидная\nДата\n ⏱ - %s\nЗадача\n 💡 - %s",
			uDate.Format("January 2"), goal.Description))

	if err != nil {
		return err
	}

	return nil
}
