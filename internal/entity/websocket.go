package entity

import (
	"github.com/gorilla/websocket"
	"sync"
)

type SubscriptionRequest struct {
	Type   string `json:"type"`
	ChatID string `json:"chatId"`
	Nonce  int    `json:"nonce"`
}

type ConnectedMessage struct {
	Connection *websocket.Conn
	Message    IncomeMessage
}

type WebSocketManager struct {
	Upgrader         websocket.Upgrader
	Subscriptions    map[*websocket.Conn]map[string]bool
	MessageQueue     chan ConnectedMessage
	SubscriptionsMux sync.Mutex
}
