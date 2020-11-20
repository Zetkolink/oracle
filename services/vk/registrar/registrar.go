package registrar

import (
	"context"

	"github.com/Zetkolink/oracle/models/keyboards"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/Zetkolink/oracle/models/whiteList"
	"github.com/Zetkolink/oracle/services"
	"github.com/go-vk-api/vk"
	"googlemaps.github.io/maps"
)

const (
	register = "register"
)

type Registrar struct {
	vkClient   *vk.Client
	mapsClient *maps.Client
	models     ModelsSet
}

type Config struct {
	VKClient   *vk.Client
	MapsClient *maps.Client
	Models     ModelsSet
}

type ModelsSet struct {
	WhiteList *whiteList.Model
	Users     *users.Model
	Keyboards *keyboards.Model
}

func NewRegistrar(config Config) *Registrar {
	return &Registrar{
		vkClient:   config.VKClient,
		mapsClient: config.MapsClient,
		models:     config.Models,
	}
}

func (r *Registrar) Handle(ctx context.Context, message services.Message) (string, error) {
	ok, err := r.models.WhiteList.Check(ctx, message.GetPeer())

	if err != nil {
		return "", err
	}

	if !ok {
		return "", nil
	}

	payload, err := message.GetPayload()

	if err != nil {
		return "", err
	}

	if payload == nil {
		err := r.SendMain(ctx, message.GetPeer())

		if err != nil {
			return "", err
		}

		return "", nil
	}

	switch payload.GetCommand() {
	case register:
		err = r.register(ctx, message.GetPeer())

		if err != nil {
			return "", err
		}

		return "menu", nil
	default:
		err := r.SendMain(ctx, message.GetPeer())

		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func (r *Registrar) SendMain(ctx context.Context, peerID int64) error {
	kb, err := r.models.Keyboards.GetKeyboard(ctx, "vk",
		"register")

	if err != nil {
		return err
	}

	err = r.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   peerID,
		"message":   "Вы готовы?",
		"random_id": 0,
		"keyboard":  kb,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (r *Registrar) register(ctx context.Context, peer int64) error {
	user, err := r.prepareUser(ctx, peer)

	if err != nil {
		return err
	}

	err = r.models.Users.Create(ctx, user)

	if err != nil {
		return err
	}

	return nil
}

func (r *Registrar) prepareUser(ctx context.Context, peer int64) (*users.User, error) {
	user, err := r.GetUser(peer)

	if err != nil {
		return nil, err
	}

	if user.City != "" {
		user.Timezone, err = r.getTimezone(ctx, user.City)
	} else {
		user.Timezone = "Asia/Yekaterinburg"
	}

	user.State = "menu"

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Registrar) GetUser(userID int64) (*users.User, error) {
	var userRaws []struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		City      struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
		} `json:"city"`
	}

	err := r.vkClient.CallMethod("users.get", vk.RequestParams{
		"user_ids": userID,
		"fields":   "city",
	}, &userRaws)

	if err != nil {
		return nil, err
	}

	raw := userRaws[0]

	return &users.User{
		ID:        userID,
		FirstName: raw.FirstName,
		LastName:  raw.LastName,
		City:      raw.City.Title,
		Active:    true,
	}, nil
}

func (r *Registrar) getTimezone(ctx context.Context, city string) (string, error) {
	geocode, err := r.getGeocode(ctx, city)

	if err != nil {
		return "", err
	}

	request := &maps.TimezoneRequest{
		Location: &geocode.Geometry.Location,
	}

	timezone, err := r.mapsClient.Timezone(ctx, request)

	if err != nil {
		return "", err
	}

	return timezone.TimeZoneID, nil
}

func (r *Registrar) getGeocode(ctx context.Context, city string) (*maps.GeocodingResult, error) {
	request := &maps.GeocodingRequest{
		Address: city,
	}

	resp, err := r.mapsClient.Geocode(ctx, request)

	if err != nil {
		return nil, err
	}

	return &resp[0], nil
}
