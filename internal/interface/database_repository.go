package repository

import (
	"Seed/internal/entity"
)

type MessagesDataBase interface {
	InsertMessage(message entity.IncomeMessage) error
	FetchHistory(chatID []byte, nonce int, amount int) ([]entity.Message, error)
}
