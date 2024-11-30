package usecase

import (
	"Seed/internal/entity"
	repository "Seed/internal/interface"
	"fmt"
	"github.com/gorilla/websocket"
)

const MessagesLimit = 100

type ResponsesUseCase struct {
	MessageRepository repository.MessageRepository
}

func (uc *ResponsesUseCase) WaitEventResponse(
	conn *websocket.Conn,
	chatID string,
) {
	outgoing := entity.WaitEventResponse{
		Type: "event",
		Event: entity.WaitEventDetail{
			Type:   "wait",
			ChatID: chatID,
		},
	}

	err := conn.WriteJSON(outgoing)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func (uc *ResponsesUseCase) NewEventResponse(
	conn *websocket.Conn,
	message entity.OutcomeMessage,
) error {
	event := entity.NewEventResponse{
		Type: "event",
		Event: entity.NewEventDetail{
			Type:    "new",
			Message: message,
		},
	}

	err := conn.WriteJSON(event)
	if err != nil {
		fmt.Println("Error sending messages:", err)
	}

	return err
}

func (uc *ResponsesUseCase) StatusResponse(conn *websocket.Conn, status bool) {
	outgoing := entity.StatusResponse{
		Type:   "response",
		Status: status,
	}

	err := conn.WriteJSON(outgoing)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func (uc *ResponsesUseCase) UnreadMessagesResponse(
	conn *websocket.Conn,
	chatID []byte,
	nonce int,
) {
	currentNonce := nonce

	for {
		messages, err := uc.MessageRepository.FetchHistory(chatID, currentNonce, MessagesLimit)

		if err != nil {
			fmt.Println("Error fetching history:", err)
			break
		}

		for _, message := range messages {
			err := uc.NewEventResponse(conn, message)

			if err != nil {
				break
			}
		}

		if len(messages) < MessagesLimit {
			break
		}

		currentNonce += MessagesLimit
	}
}