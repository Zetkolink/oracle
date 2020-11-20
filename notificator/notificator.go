package notificator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Zetkolink/oracle/models/goalTypes"
	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/go-redis/redis/v8"
)

type Notificator struct {
	Messages    chan *Message
	models      ModelsSet
	redisClient *redis.Client
}

type Config struct {
	Models      ModelsSet
	RedisClient *redis.Client
}

type ModelsSet struct {
	Users     *users.Model
	UserGoals *userGoals.Model
	GoalTypes *goalTypes.Model
}

type Message struct {
	User *users.User
	Code string
	Text string
}

func NewNotificator(config Config) *Notificator {
	return &Notificator{
		models:      config.Models,
		Messages:    make(chan *Message),
		redisClient: config.RedisClient,
	}
}

func (n *Notificator) Run() {
	go func() {
		for {
			ctx := context.Background()
			usrs, err := n.models.Users.List(ctx)

			if err != nil {
				log.Println(err)
				continue
			}

			for _, user := range usrs {
				uDate, err := user.Date(time.Now())

				if err != nil {
					log.Println(err)
					continue
				}

				nextDay := uDate.AddDate(0, 0, 1)
				nextDayGoals, err := n.models.UserGoals.ListByUserAndDate(ctx,
					user.ID, &nextDay)

				if err != nil {
					log.Println(err)
					continue
				}

				uGoals, err := n.models.UserGoals.ListByUserAndDate(ctx, user.ID, uDate)

				if err != nil {
					log.Println(err)
					continue
				}

				gTypes, err := n.models.GoalTypes.List(ctx)

				if err != nil {
					log.Println(err)
					continue
				}

				from := 12
				to := 14

				for _, uGoal := range uGoals {
					if uGoal.Type == goalTypes.Awaking {
						switch uGoal.GoalID {
						case int64(goals.Awaking6):
							from = 6
							to = 8
						case int64(goals.Awaking8):
							from = 8
							to = 10
						case int64(goals.Awaking10):
							from = 10
							to = 12
						}
					}
				}

				if uDate.Hour() >= from+10 && uDate.Hour() <= to+10 &&
					len(nextDayGoals) < len(gTypes) {

					err = n.Send(user, "next_day", "")

					if err != nil {
						log.Println(err)
					}
				}

				if len(uGoals) == 0 {
					continue
				}

				if uDate.Hour() >= from && uDate.Hour() <= to {
					err = n.Send(user, "task_list", "")

					if err != nil {
						log.Println(err)
					}
				}

				if uDate.Hour() >= from+8 && uDate.Hour() <= to+8 {
					err = n.Send(user, "mark_tasks", "")

					if err != nil {
						log.Println(err)
					}
				}
			}

			time.Sleep(1 * time.Hour)
		}
	}()
}

func (n *Notificator) Send(user *users.User, code string, text string) error {
	ok, err := n.sendCheck(user, code, text)

	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	err = n.sendMark(user, code, text)

	if err != nil {
		return err
	}

	n.Messages <- &Message{
		User: user,
		Code: code,
		Text: text,
	}

	return nil
}

func (n *Notificator) sendCheck(user *users.User, code string, text string) (bool, error) {
	err := n.redisClient.Get(context.Background(), fmt.Sprintf("%d_%s_%s",
		user.ID, code, text)).Err()

	if err != nil {
		if err == redis.Nil {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (n *Notificator) sendMark(user *users.User, code string, text string) error {
	err := n.redisClient.Set(context.Background(), fmt.Sprintf("%d_%s_%s",
		user.ID, code, text), true, 8*time.Hour).Err()

	if err != nil {
		return err
	}

	return nil
}
