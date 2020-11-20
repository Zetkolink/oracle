package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/Zetkolink/oracle/manager"
	"github.com/Zetkolink/oracle/models/evaluations"
	"github.com/Zetkolink/oracle/models/forRate"
	"github.com/Zetkolink/oracle/models/goalTypes"
	"github.com/Zetkolink/oracle/models/goals"
	"github.com/Zetkolink/oracle/models/keyboards"
	"github.com/Zetkolink/oracle/models/userGoals"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/Zetkolink/oracle/models/whiteList"
	"github.com/Zetkolink/oracle/notificator"
	"github.com/Zetkolink/oracle/observer"
	"github.com/Zetkolink/oracle/rater"
	"github.com/Zetkolink/oracle/services/vk"
	"github.com/go-redis/redis/v8"
	vkSDK "github.com/go-vk-api/vk"
	_ "github.com/lib/pq"
	"googlemaps.github.io/maps"
)

type oracle struct {
	db          *sql.DB
	models      modelSet
	vk          *vk.Service
	observer    *observer.Observer
	notificator *notificator.Notificator
	mapsClient  *maps.Client
	wg          sync.WaitGroup
}

type modelSet struct {
	users       *users.Model
	goalTypes   *goalTypes.Model
	goals       *goals.Model
	userGoals   *userGoals.Model
	evaluations *evaluations.Model
	whiteList   *whiteList.Model
}

type config struct {
	Handlers    map[string]bool
	TimezoneAPI timezoneAPIConfig
	Db          dbConfig
	Vk          vkConfig
	Cache       cacheConfig
}

type timezoneAPIConfig struct {
	Token string
}

type vkConfig struct {
	Token string
}

type dbConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type cacheConfig struct {
	Addr     string
	Password string
}

func newOracle() (*oracle, error) {
	db, err := sql.Open("postgres", cfg.Db.GetConn())

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(cfg.Cache.GetOptions())

	err = rdb.Ping(context.Background()).Err()

	if err != nil {
		return nil, err
	}

	vkClient, err := vkSDK.NewClientWithOptions(
		vkSDK.WithToken(cfg.Vk.Token),
	)

	if err != nil {
		return nil, err
	}

	usersModel, err := users.NewModel(
		users.ModelConfig{Db: db, Cache: rdb},
	)

	if err != nil {
		return nil, err
	}

	typesModel, err := goalTypes.NewModel(
		goalTypes.ModelConfig{Db: db, Cache: rdb},
	)

	if err != nil {
		return nil, err
	}

	goalsModel, err := goals.NewModel(
		goals.ModelConfig{Db: db},
	)

	if err != nil {
		return nil, err
	}

	userGoalsModel, err := userGoals.NewModel(
		userGoals.ModelConfig{Db: db},
	)

	if err != nil {
		return nil, err
	}

	evalModel, err := evaluations.NewModel(
		evaluations.ModelConfig{Db: db},
	)

	if err != nil {
		return nil, err
	}

	whiteListModel, err := whiteList.NewModel(
		whiteList.ModelConfig{Db: db},
	)

	if err != nil {
		return nil, err
	}

	forRateModel, err := forRate.NewModel(
		forRate.ModelConfig{Db: db},
	)

	if err != nil {
		return nil, err
	}

	keyboardsModel, err := keyboards.NewModel(
		keyboards.ModelConfig{Db: db},
	)

	if err != nil {
		return nil, err
	}

	mapsClient, err := maps.NewClient(
		maps.WithAPIKey(cfg.TimezoneAPI.Token),
	)

	if err != nil {
		return nil, err
	}

	mg := manager.NewManager(manager.Config{
		Models: manager.ModelsSet{
			Users:     usersModel,
			Goals:     goalsModel,
			GoalTypes: typesModel,
			UserGoals: userGoalsModel,
		}},
	)

	rt := rater.NewRater(rater.Config{
		Models: rater.ModelsSet{
			Goals:       goalsModel,
			UserGoals:   userGoalsModel,
			Evaluations: evalModel,
			ForRate:     forRateModel,
		}},
	)

	obs := observer.NewObserver(observer.Config{
		Models: observer.ModelsSet{
			UserGoals: userGoalsModel,
		}},
	)

	nt := notificator.NewNotificator(notificator.Config{
		Models: notificator.ModelsSet{
			Users:     usersModel,
			UserGoals: userGoalsModel,
			GoalTypes: typesModel,
		},
		RedisClient: rdb,
	})

	vkService := vk.NewService(vk.Config{
		VKClient:   vkClient,
		MapsClient: mapsClient,
		Models: vk.ModelsSet{
			Users:     usersModel,
			UserGoals: userGoalsModel,
			WhiteList: whiteListModel,
			Keyboards: keyboardsModel,
			GoalTypes: typesModel,
			Goals:     goalsModel,
		},
		Manager:     mg,
		Rater:       rt,
		RedisClient: rdb,
		Notificator: nt,
	})

	a := oracle{
		db:          db,
		vk:          vkService,
		observer:    obs,
		mapsClient:  mapsClient,
		notificator: nt,
		models: modelSet{
			users:       usersModel,
			goalTypes:   typesModel,
			goals:       goalsModel,
			userGoals:   userGoalsModel,
			evaluations: evalModel,
			whiteList:   whiteListModel,
		},
	}

	return &a, nil
}

func (o *oracle) Run() error {
	err := o.vk.Listen()

	if err != nil {
		return err
	}

	o.observer.Run()
	o.notificator.Run()

	return nil
}

func (o *oracle) Stop() {
	o.wg.Wait()
}

func (d *dbConfig) GetConn() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Db.Host, cfg.Db.Port, cfg.Db.User, cfg.Db.Password,
		cfg.Db.Database,
	)
}

func (c *cacheConfig) GetOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
	}
}
