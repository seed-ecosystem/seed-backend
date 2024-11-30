package usecase

import (
	"Seed/internal/entity"
	"Seed/internal/interface"
)

type MessageUseCase struct {
	MessageRepository repository.MessageRepository
}

func (uc *MessageUseCase) InsertMessage(message entity.IncomeMessage) error {
	return uc.MessageRepository.InsertMessage(message)
}

func (uc *MessageUseCase) FetchHistory(chatID []byte, nonce int, amount int) ([]entity.OutcomeMessage, error) {
	return uc.MessageRepository.FetchHistory(chatID, nonce, amount)
}

func (uc *MessageUseCase) GetLastNonce(chatID []byte) (int, error) {
	return uc.MessageRepository.GetLastNonce(chatID)
}
