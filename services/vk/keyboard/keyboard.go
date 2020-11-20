package keyboard

import (
	"encoding/json"
)

type Keyboard struct {
	OneTime bool        `json:"one_time"`
	Inline  bool        `json:"inline"`
	Buttons [][]*Button `json:"buttons"`
	Config  Config      `json:"-"`
}

type Config struct {
	OneTime bool
	Inline  bool
	Width   int
	Height  int
}

type Button struct {
	Color  string `json:"color"`
	Action Action `json:"action"`
}

type Action struct {
	Label      string  `json:"label"`
	Type       string  `json:"type"`
	Payload    Payload `json:"-"`
	PayloadStr string  `json:"payload"`
}

// Payload attachments message.
type Payload struct {
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
}

func NewKeyboard(config Config) *Keyboard {
	buttonLines := make([][]*Button, config.Height+1)

	for i := 0; i < config.Height; i++ {
		buttonLines[i] = make([]*Button, config.Width)
	}

	return &Keyboard{
		OneTime: config.OneTime,
		Inline:  config.Inline,
		Buttons: buttonLines,
		Config:  config,
	}
}

func (k *Keyboard) SetButton(i int, j int, button *Button) {
	k.Buttons[i][j] = button
}

func (k *Keyboard) DellButton(i int, j int) {
	k.Buttons[i][j] = k.Buttons[i][len(k.Buttons[i])-1]
	k.Buttons[i][len(k.Buttons[i])-1] = nil
	k.Buttons[i] = k.Buttons[i][:len(k.Buttons[i])-1]
}

func (k *Keyboard) DellButtons(i int) {
	k.Buttons[i] = k.Buttons[len(k.Buttons)-1]
	k.Buttons[len(k.Buttons)-1] = nil
	k.Buttons = k.Buttons[:len(k.Buttons)-1]
}

func (k *Keyboard) SetFooter(button *Button) {
	k.Buttons[len(k.Buttons)-1] = []*Button{button}
}

func (k *Keyboard) Marshal() (string, error) {
	for i, buttons := range k.Buttons {
	LOOP:
		for j, button := range buttons {
			if button == nil {
				k.DellButton(i, j)
				break LOOP
			}

			payload, err := json.Marshal(button.Action.Payload)

			if err != nil {
				return "", err
			}

			k.Buttons[i][j].Action.PayloadStr = string(payload)
		}
	}

	for i, buttons := range k.Buttons {
		if len(buttons) > 0 {
			if buttons[0] == nil {
				k.DellButtons(i)
			}
		}
	}

	kb, err := json.Marshal(k)

	if err != nil {
		return "", err
	}

	return string(kb), nil
}

// GetCommand get payload command.
func (p *Payload) GetCommand() string {
	return p.Command
}

// GetParam get payload param by key.
func (p *Payload) GetParam(key string) interface{} {
	_, ok := p.Params[key]

	if ok {
		return p.Params[key]
	}

	return nil
}
