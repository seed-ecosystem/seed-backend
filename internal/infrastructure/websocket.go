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
	responsesUseCase *usecase.MessagesUseCase,
	websocketUseCase *usecase.WebsocketUseCase,
) {
	conn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}

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
		case "ping":
			responsesUseCase.StatusResponse(conn, true)

		case "send":
			var sendMsg entity.IncomeMessage
			err = json.Unmarshal(msg, &sendMsg)
			if err != nil {
				fmt.Println("Error parsing 'send' message:", err)
				responsesUseCase.StatusResponse(conn, false)
				continue
			}

			message := entity.ConnectedMessage{
				Connection: conn,
				Message:    sendMsg,
			}

			select {
			case ws.MessageQueue[sendMsg.Message.ChatID] <- &message:
				fmt.Println("Message has been added to the queue for processing")
			default:
				fmt.Println("There are no subscribers to receive a message in the queue")
				responsesUseCase.StatusResponse(conn, true)
			}

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
