package repository

import (
	"Seed/internal/entity"

	"github.com/gorilla/websocket"
)

type WebsocketRepository interface {
	NewWebSocketManager() *entity.WebSocketManager
	HandleSubscribe(ws *entity.WebSocketManager, conn *websocket.Conn, chatID string)
	HandleUnsubscribe(ws *entity.WebSocketManager, conn *websocket.Conn, chatID string)
	BroadcastEvent(ws *entity.WebSocketManager, message entity.IncomeMessage)
	Disconnect(ws *entity.WebSocketManager, conn *websocket.Conn)
}
