package services

import "github.com/Zetkolink/oracle/models/users"

// Service services interface.
type Service interface {
	// Listen create stream and start listening.
	Listen() error

	// GetUser get user info from service.
	GetUser(userID int64) (*users.User, error)

	// SendMessage send message to user.
	SendMessage(peerID int64, message string) error

	// SendKeyboard send message with keyboard to user.
	SendKeyboard(peerID int64, message string, keyboard string) error

	// SendList send message with list to user.
	SendList(peerID int64, message string, list []ListItem) error
}

// Message services messages interface.
type Message interface {
	// GetPeer get message peer.
	GetPeer() int64

	// GetText get message text.
	GetText() string

	// GetPayload get message payload.
	GetPayload() (Payload, error)

	// GetPayload get message payload.
	GetUser() *users.User
}

// Payload message payload interface.
type Payload interface {
	// GetCommand get payload command.
	GetCommand() string

	// GetParam get payload param by key.
	GetParam(key string) interface{}
}

type ListItem interface {
	GetItem() interface{}
	GetLabel() string
}
