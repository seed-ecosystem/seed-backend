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
		Subscriptions: make(map[string]map[*websocket.Conn]bool),
		MessageQueue:  make(map[string]chan *entity.ConnectedMessage),
	}
}

func (uc *WebsocketUseCase) HandleSubscribe(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	ws.SubscriptionsMux.Lock()
	defer ws.SubscriptionsMux.Unlock()

	uc.addConnection(ws, conn, chatID)
}

func (uc *WebsocketUseCase) HandleUnsubscribe(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	ws.SubscriptionsMux.Lock()
	defer ws.SubscriptionsMux.Unlock()

	uc.removeConnection(ws, conn, chatID)
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

	for conn := range ws.Subscriptions[sendMsg.Message.ChatID] {
		err := uc.MessagesRepository.NewEventResponse(conn, message)
		if err != nil {
			fmt.Printf("Error broadcasting to connection: %v\n", err)
			conn.Close()
			uc.removeConnection(ws, conn, sendMsg.Message.ChatID)
		}
	}
}

func (uc *WebsocketUseCase) addConnection(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	if ws.Subscriptions[chatID] == nil {
		ws.Subscriptions[chatID] = make(map[*websocket.Conn]bool)
	}

	if ws.MessageQueue[chatID] == nil {
		uc.startMessageProcessor(ws, chatID)
	}

	ws.Subscriptions[chatID][conn] = true
}

func (uc *WebsocketUseCase) removeConnection(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	if ws.Subscriptions[chatID] != nil {
		delete(ws.Subscriptions[chatID], conn)

		if len(ws.Subscriptions[chatID]) == 0 {
			delete(ws.Subscriptions, chatID)
			uc.stopMessageProcessor(ws, chatID)
		}
	}
}

func (uc *WebsocketUseCase) startMessageProcessor(
	ws *entity.WebSocketManager,
	chatID string,
) {
	go func() {
		ws.MessageQueue[chatID] = make(chan *entity.ConnectedMessage)

		for event := range ws.MessageQueue[chatID] {
			if event == nil {
				fmt.Println("All users have unsubscribed from the chat")
				delete(ws.MessageQueue, chatID)
				return
			}

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

func (uc *WebsocketUseCase) stopMessageProcessor(
	ws *entity.WebSocketManager,
	chatID string,
) {
	ws.MessageQueue[chatID] <- nil
}
