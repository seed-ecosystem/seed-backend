package entity

import (
	"sync"

	"github.com/gorilla/websocket"
)

type SubscriptionRequest struct {
	Type   string `json:"type"`
	ChatID string `json:"queueId"`
	Nonce  int    `json:"nonce"`
}

type ConnectedMessage struct {
	Connection *websocket.Conn
	Message    IncomeMessage
}

type WebSocketManager struct {
	Upgrader         websocket.Upgrader
	Connections      map[*websocket.Conn]map[string]struct{}
	Chats            map[string]map[*websocket.Conn]struct{}
	MessageQueue     map[string]chan *ConnectedMessage
	SubscriptionsMux sync.Mutex
}
