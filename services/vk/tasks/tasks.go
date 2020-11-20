package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Zetkolink/oracle/manager"
	"github.com/Zetkolink/oracle/models/goalTypes"
	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/keyboards"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/Zetkolink/oracle/services"
	"github.com/Zetkolink/oracle/services/vk/keyboard"
	"github.com/Zetkolink/oracle/state"
	"github.com/go-redis/redis"
	"github.com/go-vk-api/vk"
	"github.com/mitchellh/mapstructure"
)

const (
	tasks = "tasks"
)

var (
	rejectBtn = &keyboard.Button{
		Color: "secondary",
		Action: keyboard.Action{
			Label: "Вернуться",
			Type:  "text",
			Payload: keyboard.Payload{
				Command: "rejectBtn",
			},
		},
	}

	errNoGoals = errors.New("goals for date not found")
)

type Tasks struct {
	vkClient    *vk.Client
	models      ModelsSet
	manager     *manager.Manager
	redisClient *redis.Client
}

type Config struct {
	VKClient    *vk.Client
	Models      ModelsSet
	Manager     *manager.Manager
	RedisClient *redis.Client
}

type ModelsSet struct {
	Users     *users.Model
	Keyboards *keyboards.Model
	GoalTypes *goalTypes.Model
	Goals     *goals.Model
}

type StateParams struct {
	Date    string `json:"date"`
	Type    int64  `json:"type"`
	Command string `json:"command"`
}

func NewTasks(config Config) *Tasks {
	return &Tasks{
		vkClient:    config.VKClient,
		models:      config.Models,
		manager:     config.Manager,
		redisClient: config.RedisClient,
	}
}

func (t *Tasks) Handle(ctx context.Context, message services.Message) (string, error) {
	payload, err := message.GetPayload()

	if err != nil {
		return "", err
	}

	st, err := state.NewState(ctx, message.GetPeer(), tasks, t.redisClient)

	if err != nil {
		return "", err
	}

	var params StateParams

	err = mapstructure.Decode(st.Params, &params)

	if err != nil {
		return "", err
	}

	var command string

	if payload != nil {
		command = payload.GetCommand()
	} else if params.Command != "" {
		command = params.Command
	}

	switch command {
	case "menu":
		err = t.models.Users.UpdateState(ctx, message.GetPeer(), "menu")

		if err != nil {
			return "", err
		}

		return "menu", nil
	case "current_tasks":
		date, err := message.GetUser().Date(time.Now())

		if err != nil {
			return "", err
		}

		err = t.MarkGoalList(ctx, message.GetUser(), *date)

		if err != nil {
			return "", err
		}
	case "update_task":
		date, err := message.GetUser().Date(time.Now())

		if err != nil {
			return "", err
		}

		err = t.MarkGoalList(ctx, message.GetUser(), *date)

		if err != nil {
			if err == errNoGoals {
				err = t.NoGoals(message.GetPeer())

				if err != nil {
					return "", err
				}

				err := t.SendMain(ctx, message.GetPeer())

				if err != nil {
					return "", err
				}

				return "", nil
			}

			return "", err
		}

		err = t.markType(ctx, *date, message.GetUser())

		if err != nil {
			return "", err
		}
	case "update_type":
		date, err := message.GetUser().Date(time.Now())

		if err != nil {
			return "", err
		}

		typeParam, ok := payload.GetParam("goal_type").(float64)

		if !ok {
			return "", errors.New("goal type not found")
		}

		gTypeID := int64(typeParam)

		err = t.manager.SetStatus(ctx, message.GetUser(), *date, gTypeID)

		err = t.MarkGoalList(ctx, message.GetUser(), *date)

		if err != nil {
			return "", err
		}

		err = t.markType(ctx, *date, message.GetUser())

		if err != nil {
			return "", err
		}
	case "observe_tasks":
		err := t.choseDate(ctx, message.GetUser(), "observe_date")

		if err != nil {
			return "", err
		}
	case "observe_date":
		dateParam := payload.GetParam("date").(string)
		date, err := time.Parse(time.RFC3339, dateParam)

		if err != nil {
			return "", err
		}

		err = t.SendGoalList(ctx, message.GetUser(), date)

		if err != nil {
			return "", err
		}
	case "change_task":
		err := t.choseDate(ctx, message.GetUser(), "change_date")

		if err != nil {
			return "", err
		}
	case "change_date":
		dateParam := payload.GetParam("date").(string)
		date, err := time.Parse(time.RFC3339, dateParam)

		if err != nil {
			return "", err
		}

		err = t.SendGoalList(ctx, message.GetUser(), date)

		if err != nil {
			return "", err
		}

		err = t.choseType(ctx, date, message.GetUser())

		if err != nil {
			return "", err
		}
	case "change_type":
		dateParam, ok := payload.GetParam("date").(string)

		if !ok {
			return "", errors.New("date not found")
		}

		date, err := time.Parse(time.RFC3339, dateParam)

		if err != nil {
			return "", err
		}

		typeParam, ok := payload.GetParam("goal_type").(float64)

		if !ok {
			return "", errors.New("goal type not found")
		}

		gTypeID := int64(typeParam)

		params.Type = gTypeID
		params.Date = date.Format(time.RFC3339)
		err = st.SetParams(ctx, params)

		if err != nil {
			return "", err
		}

		gType, err := t.models.GoalTypes.Get(ctx, gTypeID)

		if err != nil {
			return "", err
		}

		uGoal, err := t.manager.GetByType(ctx, message.GetUser(), date, gTypeID)

		if err != nil {
			return "", err
		}

		if uGoal != nil {
			err = t.SendGoal(ctx, message.GetPeer(), uGoal)

			if err != nil {
				return "", err
			}
		}

		if gType.FromList {
			err = t.choseGoal(ctx, message.GetPeer(), gType)

			if err != nil {
				return "", err
			}

			return "", nil
		}

		params.Command = "input_goal"
		err = st.SetParams(ctx, params)

		if err != nil {
			return "", err
		}

		err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
			"peer_id":   message.GetPeer(),
			"message":   "Введите",
			"random_id": 0,
		}, nil)

		if err != nil {
			return "", err
		}

		return "", nil
	case "chose_goal":
		if params.Type == 0 || params.Date == "" {
			return "", errors.New("not found params")
		}

		goalParam, ok := payload.GetParam("goal").(float64)

		if !ok {
			return "", errors.New("goal type not found")
		}

		goal, err := t.models.Goals.Get(ctx, int64(goalParam))

		if err != nil {
			return "", err
		}

		date, err := time.Parse(time.RFC3339, params.Date)

		if err != nil {
			return "", err
		}

		_, err = t.manager.AssignGoal(ctx, message.GetUser(), goal, date)

		if err != nil {
			return "", err
		}

		err = t.SendGoalList(ctx, message.GetUser(), date)

		if err != nil {
			return "", err
		}

		st.Clear(ctx)

		err = t.choseType(ctx, date, message.GetUser())

		if err != nil {
			return "", err
		}

		return "", nil
	case "input_goal":
		if params.Type == 0 || params.Date == "" {
			return "", errors.New("not found params")
		}

		goal := &goals.Goal{
			Type:        params.Type,
			Description: message.GetText(),
		}

		date, err := time.Parse(time.RFC3339, params.Date)

		if err != nil {
			return "", err
		}

		_, err = t.manager.AssignGoal(ctx, message.GetUser(), goal, date)

		if err != nil {
			return "", err
		}

		err = t.SendGoalList(ctx, message.GetUser(), date)

		if err != nil {
			return "", err
		}

		st.Clear(ctx)

		err = t.choseType(ctx, date, message.GetUser())

		if err != nil {
			return "", err
		}

		return "", nil
	default:
		err := t.SendMain(ctx, message.GetPeer())

		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func (t *Tasks) SendMain(ctx context.Context, peerID int64) error {
	kb, err := t.models.Keyboards.GetKeyboard(ctx, "vk",
		"tasks")

	if err != nil {
		return err
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   peerID,
		"message":   "Задачи",
		"random_id": 0,
		"keyboard":  kb,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) NoGoals(peerID int64) error {
	err := t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   peerID,
		"message":   "Вы ничего не запланировали на этот день",
		"random_id": 0,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) SendGoal(ctx context.Context, peerID int64, uGoal *userGoals.UserGoal) error {
	goal, err := t.models.Goals.Get(ctx, uGoal.GoalID)

	if err != nil {
		return err
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   peerID,
		"message":   fmt.Sprintf("Текущая задача\n - %s", goal.Description),
		"random_id": 0,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) SendGoalList(ctx context.Context, user *users.User, date time.Time) error {
	uGoals, err := t.manager.UserGoals(ctx, user, date)

	if err != nil {
		return err
	}

	uGoalsMap := make(map[int64]*userGoals.UserGoal)

	for _, uGoal := range uGoals {
		uGoalsMap[uGoal.Type] = uGoal
	}

	types, err := t.models.GoalTypes.List(ctx)

	if err != nil {
		return err
	}

	var message string

	for _, gType := range types {
		uGoal, ok := uGoalsMap[gType.ID]

		if ok {
			goal, err := t.models.Goals.Get(ctx, uGoal.GoalID)

			if err != nil {
				return err
			}

			message += fmt.Sprintf("%s\n 💡 %s\n\n", gType.Name, goal.Description)
		} else {
			message += fmt.Sprintf("%s\n 📍 Не запланировано\n\n", gType.Name)
		}
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   message,
		"random_id": 0,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) MarkGoalList(ctx context.Context, user *users.User, date time.Time) error {
	uGoals, err := t.manager.UserGoals(ctx, user, date)

	if err != nil {
		return err
	}

	uGoalsMap := make(map[int64]*userGoals.UserGoal)

	for _, uGoal := range uGoals {
		uGoalsMap[uGoal.Type] = uGoal
	}

	types, err := t.models.GoalTypes.List(ctx)

	if err != nil {
		return err
	}

	var message string

	for _, gType := range types {
		uGoal, ok := uGoalsMap[gType.ID]

		if ok {
			goal, err := t.models.Goals.Get(ctx, uGoal.GoalID)

			if err != nil {
				return err
			}

			var mark string

			switch uGoal.Status {
			case userGoals.StatusInProgress:
				mark = "🎯 В процессе"
			case userGoals.StatusComplete:
				mark = "🍏 Выполнено"
			case userGoals.StatusSoon:
				mark = "📝 Запланировано"
			case userGoals.StatusFailed:
				mark = "🍎 Првоалено"
			}

			message += fmt.Sprintf("%s\n 💡 %s\nСтатус - %s\n\n",
				gType.Name, goal.Description, mark)
		}
	}

	if message == "" {
		return errNoGoals
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   message,
		"random_id": 0,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) SendMarkList(ctx context.Context, user *users.User, date time.Time) error {
	uGoals, err := t.manager.UserGoals(ctx, user, date)

	if err != nil {
		return err
	}

	uGoalsMap := make(map[int64]*userGoals.UserGoal)

	for _, uGoal := range uGoals {
		uGoalsMap[uGoal.Type] = uGoal
	}

	types, err := t.models.GoalTypes.List(ctx)

	if err != nil {
		return err
	}

	var message string

	for _, gType := range types {
		uGoal, ok := uGoalsMap[gType.ID]

		if ok {
			goal, err := t.models.Goals.Get(ctx, uGoal.GoalID)

			if err != nil {
				return err
			}

			message += fmt.Sprintf("%s\n 💡 %s\n\n", gType.Name, goal.Description)
		} else {
			message += fmt.Sprintf("%s\n 📍 Не запланировано\n\n", gType.Name)
		}
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   message,
		"random_id": 0,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) choseDate(ctx context.Context, user *users.User, command string) error {
	userTime, err := user.Date(time.Now())

	if err != nil {
		return err
	}

	kb := keyboard.NewKeyboard(keyboard.Config{
		OneTime: false,
		Inline:  false,
		Width:   3,
		Height:  2,
	})

	c := 0

	for i := 0; i < kb.Config.Height; i++ {
		for j := 0; j < kb.Config.Width; j++ {
			date := userTime.AddDate(0, 0, c+1)

			btn := &keyboard.Button{
				Color: "primary",
				Action: keyboard.Action{
					Label: fmt.Sprintf("%d %s", date.Day(), date.Month()),
					Type:  "text",
					Payload: keyboard.Payload{
						Command: command,
						Params: map[string]interface{}{
							"date": date,
						},
					},
				},
			}

			ok, err := t.manager.CheckDate(ctx, user, date)

			if err != nil {
				return err
			}

			if ok {
				btn.Color = "positive"
			}

			kb.SetButton(i, j, btn)
			c++
		}
	}

	kb.SetFooter(rejectBtn)
	kbStr, err := kb.Marshal()

	if err != nil {
		return err
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   "Выберите дату",
		"random_id": 0,
		"keyboard":  kbStr,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) markType(ctx context.Context, date time.Time, user *users.User) error {
	uGoals, err := t.manager.UserGoals(ctx, user, date)

	if err != nil {
		return err
	}

	uGoalsMap := make(map[int64]*userGoals.UserGoal)

	for _, uGoal := range uGoals {
		uGoalsMap[uGoal.Type] = uGoal
	}

	types, err := t.models.GoalTypes.List(ctx)

	if err != nil {
		return err
	}

	kb := keyboard.NewKeyboard(keyboard.Config{
		OneTime: false,
		Inline:  false,
		Width:   2,
		Height:  len(types) / 2,
	})

	c := 0

	for i := 0; i < kb.Config.Height; i++ {
		for j := 0; j < kb.Config.Width; j++ {
			btn := &keyboard.Button{
				Color: "primary",
				Action: keyboard.Action{
					Label: types[c].Name,
					Type:  "text",
					Payload: keyboard.Payload{
						Command: "update_type",
						Params: map[string]interface{}{
							"goal_type": types[c].ID,
							"date":      date.Format(time.RFC3339),
						},
					},
				},
			}

			if uGoalsMap[types[c].ID] != nil &&
				uGoalsMap[types[c].ID].Status == userGoals.StatusComplete {

				btn.Color = "positive"
			}

			kb.SetButton(i, j, btn)
			c++
		}
	}

	kb.SetFooter(rejectBtn)
	kbStr, err := kb.Marshal()

	if err != nil {
		return err
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   "Отметьте выполненные",
		"random_id": 0,
		"keyboard":  kbStr,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) choseType(ctx context.Context, date time.Time, user *users.User) error {
	types, err := t.models.GoalTypes.List(ctx)

	if err != nil {
		return err
	}

	kb := keyboard.NewKeyboard(keyboard.Config{
		OneTime: true,
		Inline:  false,
		Width:   2,
		Height:  len(types) / 2,
	})

	c := 0

	for i := 0; i < kb.Config.Height; i++ {
		for j := 0; j < kb.Config.Width; j++ {
			btn := &keyboard.Button{
				Color: "primary",
				Action: keyboard.Action{
					Label: types[c].Name,
					Type:  "text",
					Payload: keyboard.Payload{
						Command: "change_type",
						Params: map[string]interface{}{
							"goal_type": types[c].ID,
							"date":      date.Format(time.RFC3339),
						},
					},
				},
			}

			goal, err := t.manager.GetByType(ctx, user, date, types[c].ID)

			if err != nil {
				return err
			}

			if goal != nil {
				btn.Color = "positive"
			}

			kb.SetButton(i, j, btn)
			c++
		}
	}

	kb.SetFooter(rejectBtn)
	kbStr, err := kb.Marshal()

	if err != nil {
		return err
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   user.ID,
		"message":   "Выберите тип",
		"random_id": 0,
		"keyboard":  kbStr,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (t *Tasks) choseGoal(ctx context.Context, peerID int64, gType *goalTypes.GoalType) error {
	gls, err := t.models.Goals.List(ctx, gType.ID)

	if err != nil {
		return err
	}

	var h int

	if len(gls)%2 > 0 {
		h = len(gls)/2 + 1
	} else {
		h = len(gls) / 2
	}

	kb := keyboard.NewKeyboard(keyboard.Config{
		OneTime: false,
		Inline:  false,
		Width:   2,
		Height:  h,
	})

	c := 0

	for i := 0; i < kb.Config.Height; i++ {
		for j := 0; j < kb.Config.Width; j++ {
			if len(gls) < c+1 {
				continue
			}

			btn := &keyboard.Button{
				Color: "primary",
				Action: keyboard.Action{
					Label: gls[c].Description,
					Type:  "text",
					Payload: keyboard.Payload{
						Command: "chose_goal",
						Params: map[string]interface{}{
							"goal": gls[c].ID,
						},
					},
				},
			}

			kb.SetButton(i, j, btn)
			c++
		}
	}

	kb.SetFooter(rejectBtn)
	kbStr, err := kb.Marshal()

	if err != nil {
		return err
	}

	err = t.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   peerID,
		"message":   "Выберите задачу",
		"random_id": 0,
		"keyboard":  kbStr,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}
