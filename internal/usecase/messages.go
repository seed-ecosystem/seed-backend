package usecase

import (
	"Seed/internal/entity"
	repository "Seed/internal/interface"
	"encoding/base64"
	"fmt"

	"github.com/gorilla/websocket"
)

const MessagesLimit = 100

type MessagesUseCase struct {
	MessagesDataBase repository.MessagesDataBase
}

func (uc *MessagesUseCase) WaitEventResponse(
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

func (uc *MessagesUseCase) NewEventResponse(
	conn *websocket.Conn,
	message entity.Message,
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

func (uc *MessagesUseCase) StatusResponse(
	conn *websocket.Conn,
	status bool,
) {
	outgoing := entity.StatusResponse{
		Type:   "response",
		Status: status,
	}

	err := conn.WriteJSON(outgoing)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func (uc *MessagesUseCase) UnreadMessagesResponse(
	conn *websocket.Conn,
	chatID []byte,
	nonce int,
) {
	currentNonce := nonce

	for {
		messages, err := uc.FetchHistory(chatID, currentNonce, MessagesLimit)

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

func (uc *MessagesUseCase) IsValidMessage(
	message entity.IncomeMessage,
) bool {
	chatID, err := base64.StdEncoding.DecodeString(message.Message.ChatID)
	if err != nil || len(chatID) != 32 {
		fmt.Println("Invalid ChatID")
		return false
	}

	signature, err := base64.StdEncoding.DecodeString(message.Message.Signature)
	if err != nil || len(signature) != 32 {
		fmt.Println("Invalid Signature")
		return false
	}

	contentIV, err := base64.StdEncoding.DecodeString(message.Message.ContentIV)
	if err != nil || len(contentIV) != 12 {
		fmt.Println("Invalid ContentIV")
		return false
	}

	return true
}

func (uc *MessagesUseCase) InsertMessage(
	message entity.IncomeMessage,
) error {
	return uc.MessagesDataBase.InsertMessage(message)
}

func (uc *MessagesUseCase) FetchHistory(
	chatID []byte,
	nonce int,
	amount int,
) ([]entity.Message, error) {
	return uc.MessagesDataBase.FetchHistory(chatID, nonce, amount)
}
