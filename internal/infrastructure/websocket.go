package infrastructure

import (
	"Seed/internal/entity"
	"Seed/internal/usecase"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

func HandleWebSocketConnection(
	ws *entity.WebSocketManager,
	w http.ResponseWriter,
	r *http.Request,
	messageUseCase *usecase.MessageUseCase,
	responsesUseCase *usecase.ResponsesUseCase,
	websocketUseCase *usecase.WebsocketUseCase,
) {
	conn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer func() {
		ws.SubscriptionsMux.Lock()
		delete(ws.Subscriptions, conn)
		ws.SubscriptionsMux.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		var incoming map[string]interface{}
		err = json.Unmarshal(msg, &incoming)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			continue
		}

		switch incoming["type"] {
		case "send":
			var sendMsg entity.IncomeMessage
			err = json.Unmarshal(msg, &sendMsg)
			if err != nil {
				fmt.Println("Error parsing 'send' message:", err)
				responsesUseCase.StatusResponse(conn, false)
				continue
			}

			err = messageUseCase.InsertMessage(sendMsg)
			if err != nil {
				fmt.Println("Error inserting message:", err)
				responsesUseCase.StatusResponse(conn, false)
				continue
			}

			message := entity.OutcomeMessage{
				Nonce:     sendMsg.Message.Nonce,
				ChatID:    sendMsg.Message.ChatID,
				Signature: sendMsg.Message.Signature,
				Content:   sendMsg.Message.Content,
				ContentIV: sendMsg.Message.ContentIV,
			}

			responsesUseCase.StatusResponse(conn, true)
			websocketUseCase.BroadcastEvent(ws.Subscriptions, message)

		case "subscribe":
			var subRequest entity.SubscriptionRequest
			err = json.Unmarshal(msg, &subRequest)
			if err != nil {
				fmt.Println("Error parsing 'subscribe' request:", err)
				responsesUseCase.StatusResponse(conn, false)
				continue
			}

			chatID, err := base64.StdEncoding.DecodeString(subRequest.ChatID)
			if err != nil {
				fmt.Println("Invalid ChatID in 'subscribe' request:", err)
				responsesUseCase.StatusResponse(conn, false)
				continue
			}

			websocketUseCase.HandleSubscribe(ws, conn, subRequest.ChatID)
			responsesUseCase.StatusResponse(conn, true)
			responsesUseCase.UnreadMessagesResponse(conn, chatID, subRequest.Nonce)
			responsesUseCase.WaitEventResponse(conn, subRequest.ChatID)

		case "unsubscribe":
			var unsubRequest entity.SubscriptionRequest
			err = json.Unmarshal(msg, &unsubRequest)
			if err != nil {
				fmt.Println("Error parsing 'unsubscribe' request:", err)
				responsesUseCase.StatusResponse(conn, false)
				continue
			}

			websocketUseCase.HandleUnsubscribe(ws, conn, unsubRequest.ChatID)
			responsesUseCase.StatusResponse(conn, true)

		default:
			fmt.Println("Unknown request type:", incoming["type"])
			responsesUseCase.StatusResponse(conn, false)
		}
	}
}
