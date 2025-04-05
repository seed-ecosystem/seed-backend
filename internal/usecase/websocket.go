package usecase

import (
	"Seed/internal/entity"
	repository "Seed/internal/interface"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebsocketUseCase struct {
	MessagesRepository repository.MessagesRepository
}

func (uc *WebsocketUseCase) NewWebSocketManager() *entity.WebSocketManager {
	return &entity.WebSocketManager{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Chats:        make(map[string]map[*websocket.Conn]struct{}),
		Connections:  make(map[*websocket.Conn]map[string]struct{}),
		MessageQueue: make(map[string]chan *entity.ConnectedMessage),
	}
}

func (uc *WebsocketUseCase) HandleSubscribe(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	uc.subscribeToChat(ws, conn, chatID)
}

func (uc *WebsocketUseCase) HandleUnsubscribe(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	uc.unsubscribeFromChat(ws, conn, chatID)
}

func (uc *WebsocketUseCase) BroadcastEvent(
	ws *entity.WebSocketManager,
	sendMsg entity.IncomeMessage,
) {
	message := entity.Message{
		Nonce:     sendMsg.Message.Nonce,
		ChatID:    sendMsg.Message.ChatID,
		Signature: sendMsg.Message.Signature,
		Content:   sendMsg.Message.Content,
		ContentIV: sendMsg.Message.ContentIV,
	}

	for conn := range ws.Chats[sendMsg.Message.ChatID] {
		err := uc.MessagesRepository.NewEventResponse(conn, message)
		if err != nil {
			fmt.Printf("Error broadcasting to connection: %v\n", err)
			uc.Disconnect(ws, conn)
		}
	}
}

func (uc *WebsocketUseCase) Disconnect(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
) {
	defer conn.Close()

	if ws.Connections[conn] != nil {
		for chatID := range ws.Connections[conn] {
			uc.unsubscribeFromChat(ws, conn, chatID)
		}

		delete(ws.Connections, conn)
	}
}

func (uc *WebsocketUseCase) subscribeToChat(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	ws.SubscriptionsMux.Lock()
	defer ws.SubscriptionsMux.Unlock()

	if ws.Connections[conn] == nil {
		ws.Connections[conn] = make(map[string]struct{})
	}

	if ws.Chats[chatID] == nil {
		ws.Chats[chatID] = make(map[*websocket.Conn]struct{})
	}

	if ws.MessageQueue[chatID] == nil {
		uc.startMessageProcessor(ws, chatID)
	}

	ws.Connections[conn][chatID] = struct{}{}
	ws.Chats[chatID][conn] = struct{}{}
}

func (uc *WebsocketUseCase) unsubscribeFromChat(
	ws *entity.WebSocketManager,
	conn *websocket.Conn,
	chatID string,
) {
	ws.SubscriptionsMux.Lock()
	defer ws.SubscriptionsMux.Unlock()

	if ws.Connections[conn] != nil {
		delete(ws.Connections[conn], chatID)

		if len(ws.Connections[conn]) == 0 {
			delete(ws.Connections, conn)
		}
	}

	if ws.Chats[chatID] != nil {
		delete(ws.Chats[chatID], conn)

		if len(ws.Chats[chatID]) == 0 {
			delete(ws.Chats, chatID)
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

			err := uc.MessagesRepository.InsertMessage(event.Message)
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
