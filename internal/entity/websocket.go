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

type WebSocketManager struct {
	Upgrader         websocket.Upgrader
	Subscriptions    map[*websocket.Conn]map[string]bool
	SubscriptionsMux sync.Mutex
}
