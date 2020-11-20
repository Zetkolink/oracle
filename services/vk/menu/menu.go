package menu

import (
	"context"

	"github.com/Zetkolink/oracle/models/keyboards"
	"github.com/Zetkolink/oracle/models/users"
	"github.com/Zetkolink/oracle/models/whiteList"
	"github.com/Zetkolink/oracle/services"
	"github.com/go-vk-api/vk"
)

const (
	menu = "menu"
)

type Menu struct {
	vkClient *vk.Client
	models   ModelsSet
}

type Config struct {
	VKClient *vk.Client
	Models   ModelsSet
}

type ModelsSet struct {
	WhiteList *whiteList.Model
	Users     *users.Model
	Keyboards *keyboards.Model
}

func NewMenu(config Config) *Menu {
	return &Menu{
		vkClient: config.VKClient,
		models:   config.Models,
	}
}

func (m *Menu) Handle(ctx context.Context, message services.Message) (string, error) {
	payload, err := message.GetPayload()

	if err != nil {
		return "", err
	}

	if payload == nil {
		err := m.SendMain(ctx, message.GetPeer())

		if err != nil {
			return "", err
		}

		return "", nil
	}

	switch payload.GetCommand() {
	case "to_tasks":
		err = m.models.Users.UpdateState(ctx, message.GetPeer(), "tasks")

		if err != nil {
			return "", err
		}

		return "tasks", nil
	case "to_rate":
		err = m.models.Users.UpdateState(ctx, message.GetPeer(), "rate")

		if err != nil {
			return "", err
		}

		return "rate", nil
	default:
		err := m.SendMain(ctx, message.GetPeer())

		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func (m *Menu) SendMain(ctx context.Context, peerID int64) error {
	kb, err := m.models.Keyboards.GetKeyboard(ctx, "vk",
		"menu")

	if err != nil {
		return err
	}

	err = m.vkClient.CallMethod("messages.send", vk.RequestParams{
		"peer_id":   peerID,
		"message":   "Главное меню",
		"random_id": 0,
		"keyboard":  kb,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}
