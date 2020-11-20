package vk

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Zetkolink/oracle/manager"
	"github.com/Zetkolink/oracle/models/goalTypes"
	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/keyboards"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/Zetkolink/oracle/models/whiteList"
	"github.com/Zetkolink/oracle/notificator"
	"github.com/Zetkolink/oracle/rater"
	s "github.com/Zetkolink/oracle/services"
	"github.com/Zetkolink/oracle/services/vk/appraiser"
	"github.com/Zetkolink/oracle/services/vk/keyboard"
	"github.com/Zetkolink/oracle/services/vk/menu"
	"github.com/Zetkolink/oracle/services/vk/registrar"
	"github.com/Zetkolink/oracle/services/vk/tasks"
	"github.com/go-redis/redis/v8"
	vkSDK "github.com/go-vk-api/vk"
	lp "github.com/go-vk-api/vk/longpoll/user"
	"googlemaps.github.io/maps"
)

// Service wrapper for vk api client.
type Service struct {
	*vkSDK.Client
	redisClient *redis.Client
	models      ModelsSet
	mapsClient  *maps.Client
	registrar   *registrar.Registrar
	menu        *menu.Menu
	tasks       *tasks.Tasks
	appraiser   *appraiser.Appraiser
	notificator *notificator.Notificator
}

// Config configuration for Service.
type Config struct {
	Models      ModelsSet
	VKClient    *vkSDK.Client
	RedisClient *redis.Client
	MapsClient  *maps.Client
	Manager     *manager.Manager
	Rater       *rater.Rater
	Notificator *notificator.Notificator
}

type ModelsSet struct {
	Users     *users.Model
	WhiteList *whiteList.Model
	Keyboards *keyboards.Model
	GoalTypes *goalTypes.Model
	Goals     *goals.Model
	UserGoals *userGoals.Model
}

// Message wrapper for vk new message.
type Message struct {
	*lp.NewMessage
	user *users.User
}

// NewService create new instance of Service.
func NewService(config Config) *Service {
	r := registrar.NewRegistrar(registrar.Config{
		VKClient:   config.VKClient,
		MapsClient: config.MapsClient,
		Models: registrar.ModelsSet{
			WhiteList: config.Models.WhiteList,
			Users:     config.Models.Users,
			Keyboards: config.Models.Keyboards,
		},
	})

	m := menu.NewMenu(menu.Config{
		VKClient: config.VKClient,
		Models: menu.ModelsSet{
			WhiteList: config.Models.WhiteList,
			Users:     config.Models.Users,
			Keyboards: config.Models.Keyboards,
		},
	})

	t := tasks.NewTasks(tasks.Config{
		VKClient: config.VKClient,
		Models: tasks.ModelsSet{
			Users:     config.Models.Users,
			Keyboards: config.Models.Keyboards,
			GoalTypes: config.Models.GoalTypes,
			Goals:     config.Models.Goals,
		},
		Manager:     config.Manager,
		RedisClient: config.RedisClient,
	})

	a := appraiser.NewAppraiser(appraiser.Config{
		VKClient: config.VKClient,
		Rater:    config.Rater,
		Models: appraiser.ModelsSet{
			Users:     config.Models.Users,
			UserGoals: config.Models.UserGoals,
			Goals:     config.Models.Goals,
		},
		Notificator: config.Notificator,
	})

	return &Service{
		Client:      config.VKClient,
		mapsClient:  config.MapsClient,
		models:      config.Models,
		notificator: config.Notificator,
		registrar:   r,
		menu:        m,
		tasks:       t,
		appraiser:   a,
	}
}

// Listen create stream and start listening.
func (s *Service) Listen() error {
	s.listenNotificator()
	stream, err := s.createStream()
	ctx := context.Background()

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case update, ok := <-stream.Updates:
				if !ok {
					log.Println("error listen")
				}

				switch msg := update.Data.(type) {
				case *lp.NewMessage:
					_, ok := s.parseFlags(msg.Flags)[2]

					if ok {
						continue
					}

					user, err := s.models.Users.Get(ctx, msg.PeerID)

					if err != nil {
						log.Println(err)
						continue
					}

					var state string

					if user == nil {
						state, err = s.registrar.Handle(ctx,
							&Message{
								NewMessage: msg,
								user:       user,
							},
						)

						if err != nil {
							log.Println(err)
						}
					} else {
						state = user.State
					}

					for state != "" {
						switch state {
						case "menu":
							state, err = s.menu.Handle(ctx, &Message{
								NewMessage: msg,
								user:       user,
							})

							if err != nil {
								log.Println(err)
							}
						case "tasks":
							state, err = s.tasks.Handle(ctx, &Message{
								NewMessage: msg,
								user:       user,
							})

							if err != nil {
								log.Println(err)
							}
						case "rate":
							state, err = s.appraiser.Handle(ctx, &Message{
								NewMessage: msg,
								user:       user,
							})

							if err != nil {
								log.Println(err)
							}
						default:
							state = ""
						}
					}
				}
			case err, _ := <-stream.Errors:
				stream, err = s.createStream()

				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	return nil
}

func (s *Service) listenNotificator() {
	go func() {
		for {
			message := <-s.notificator.Messages

			switch message.Code {
			case "disapprove":
				err := s.CallMethod("messages.send", vkSDK.RequestParams{
					"peer_id":   message.User.ID,
					"message":   message.Text,
					"random_id": 0,
				}, nil)

				if err != nil {
					log.Println(err)
				}
			case "next_day":
				err := s.CallMethod("messages.send", vkSDK.RequestParams{
					"peer_id":   message.User.ID,
					"message":   "Не забудьте создать список задач на зватрашний день",
					"random_id": 0,
				}, nil)

				if err != nil {
					log.Println(err)
				}
			case "task_list":
				uDate, err := message.User.Date(time.Now())

				if err != nil {
					log.Println(err)
					continue
				}

				err = s.tasks.SendGoalList(context.Background(), message.User, *uDate)

				if err != nil {
					log.Println(err)
					continue
				}

				err = s.CallMethod("messages.send", vkSDK.RequestParams{
					"peer_id":   message.User.ID,
					"message":   "Доброе утро! Ваш список задач на сегодня",
					"random_id": 0,
				}, nil)

				if err != nil {
					log.Println(err)
				}
			case "mark_tasks":
				ctx := context.Background()
				uDate, err := message.User.Date(time.Now())

				if err != nil {
					log.Println(err)
					continue
				}

				err = s.tasks.MarkGoalList(ctx, message.User, *uDate)

				if err != nil {
					log.Println(err)
					continue
				}

				err = s.models.Users.UpdateState(ctx, message.User.ID, "tasks")

				if err != nil {
					log.Println(err)
					continue
				}

				kb, err := s.models.Keyboards.GetKeyboard(ctx, "vk", "tasks")

				if err != nil {
					log.Println(err)
					continue
				}

				err = s.CallMethod("messages.send", vkSDK.RequestParams{
					"peer_id":   message.User.ID,
					"message":   "Доброе вечер! Обновите статусы задач",
					"random_id": 0,
					"keyboard":  kb,
				}, nil)

				if err != nil {
					log.Println(err)
				}
			}
		}
	}()
}

func (s *Service) createStream() (*lp.Stream, error) {
	client, err := lp.NewWithOptions(s.Client,
		lp.WithMode(lp.ReceiveAttachments))

	if err != nil {
		return nil, err
	}

	stream, err := client.GetUpdatesStream(0)

	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *Service) parseFlags(flagsRaw int64) map[int64]struct{} {
	flags := make(map[int64]struct{})
	bit := int64(1)

	for i := 0; i < 31; i++ {
		if flagsRaw&bit > 0 {
			flags[bit] = struct{}{}
		}

		bit *= 2
	}

	return flags
}

// GetPeer get message peer.
func (m *Message) GetPeer() int64 {
	return m.PeerID
}

// GetText get message text.
func (m *Message) GetText() string {
	return m.Text
}

// GetPayload get message payload.
func (m *Message) GetPayload() (s.Payload, error) {
	payloadValue := m.Attachments["payload"]

	if payloadValue != "" {
		var payload keyboard.Payload

		err := json.Unmarshal([]byte(payloadValue), &payload)

		if err != nil {
			return nil, err
		}

		return &payload, nil
	}

	return nil, nil
}

// GetUser get message user.
func (m *Message) GetUser() *users.User {
	return m.user
}
