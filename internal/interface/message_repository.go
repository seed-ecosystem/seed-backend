package repository

import (
	"Seed/internal/entity"
)

type MessageRepository interface {
	InsertMessage(message entity.IncomeMessage) error
	FetchHistory(chatID []byte, nonce int, amount int) ([]entity.OutcomeMessage, error)
	GetLastNonce(chatID []byte) (int, error)
}
