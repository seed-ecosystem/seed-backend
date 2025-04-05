package repository

import (
	"Seed/internal/entity"

	"github.com/gorilla/websocket"
)

type MessagesRepository interface {
	WaitEventResponse(conn *websocket.Conn, chatID string)
	NewEventResponse(conn *websocket.Conn, message entity.Message) error

	StatusResponse(conn *websocket.Conn, status bool)
	UnreadMessagesResponse(conn *websocket.Conn, chatID []byte, nonce int)

	IsValidMessage(message entity.IncomeMessage) bool

	MessagesDataBase
}
