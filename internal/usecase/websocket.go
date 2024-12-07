package usecase

import (
	"Seed/internal/entity"
	repository "Seed/internal/interface"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

type WebsocketUseCase struct {
	MessagesRepository repository.MessagesRepository
	DataBaseRepository repository.DataBaseRepository
}

func (uc *WebsocketUseCase) NewWebSocketManager() *entity.WebSocketManager {
	return &entity.WebSocketManager{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Subscriptions: make(map[*websocket.Conn]map[string]bool),
		MessageQueue:  make(chan entity.ConnectedMessage),
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
	ws *entity.WebSocketManager,
	sendMsg entity.IncomeMessage,
) {
	message := entity.OutcomeMessage{
		Nonce:     sendMsg.Message.Nonce,
		ChatID:    sendMsg.Message.ChatID,
		Signature: sendMsg.Message.Signature,
		Content:   sendMsg.Message.Content,
		ContentIV: sendMsg.Message.ContentIV,
	}

	for conn, chatIDs := range ws.Subscriptions {
		if chatIDs[message.ChatID] {
			err := uc.MessagesRepository.NewEventResponse(conn, message)
			if err != nil {
				fmt.Printf("Error broadcasting to connection: %v\n", err)
				conn.Close()
				delete(ws.Subscriptions, conn)
			}
		}
	}
}

func (uc *WebsocketUseCase) StartMessageProcessor(
	ws *entity.WebSocketManager,
) {
	go func() {
		for event := range ws.MessageQueue {
			err := uc.DataBaseRepository.InsertMessage(event.Message)
			if err != nil {
				fmt.Println("Error inserting message:", err)
				uc.MessagesRepository.StatusResponse(event.Connection, false)
				continue
			}

			uc.MessagesRepository.StatusResponse(event.Connection, true)
			uc.BroadcastEvent(ws, event.Message)
		}
	}()
}
