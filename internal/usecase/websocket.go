package usecase

import (
	"Seed/internal/entity"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

type WebsocketUseCase struct{}

func (uc *WebsocketUseCase) NewWebSocketManager() *entity.WebSocketManager {
	return &entity.WebSocketManager{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Subscriptions: make(map[*websocket.Conn]map[string]bool),
	}
}

func (uc *WebsocketUseCase) HandleSubscribe(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	ws.SubscriptionsMux.Lock()
	defer ws.SubscriptionsMux.Unlock()

	if ws.Subscriptions[conn] == nil {
		ws.Subscriptions[conn] = make(map[string]bool)
	}

	ws.Subscriptions[conn][chatID] = true
}

func (uc *WebsocketUseCase) HandleUnsubscribe(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	ws.SubscriptionsMux.Lock()
	defer ws.SubscriptionsMux.Unlock()

	if ws.Subscriptions[conn] != nil {
		delete(ws.Subscriptions[conn], chatID)

		if len(ws.Subscriptions[conn]) == 0 {
			delete(ws.Subscriptions, conn)
		}
	}
}

func (uc *WebsocketUseCase) BroadcastEvent(
	subscriptions map[*websocket.Conn]map[string]bool,
	message entity.OutcomeMessage,
) {
	event := entity.NewEventResponse{
		Type: "event",
		Event: entity.NewEventDetail{
			Type:    "new",
			Message: message,
		},
	}

	for conn, chatIDs := range subscriptions {
		if chatIDs[message.ChatID] {
			err := conn.WriteJSON(event)
			if err != nil {
				fmt.Printf("Error broadcasting to connection: %v\n", err)
				conn.Close()
				delete(subscriptions, conn)
			}
		}
	}
}
