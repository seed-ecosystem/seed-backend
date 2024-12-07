package usecase

import (
	"Seed/internal/entity"
	"Seed/internal/interface"
)

type DataBaseUseCase struct {
	MessageRepository repository.DataBaseRepository
}

func (uc *DataBaseUseCase) InsertMessage(message entity.IncomeMessage) error {
	return uc.MessageRepository.InsertMessage(message)
}

func (uc *DataBaseUseCase) FetchHistory(chatID []byte, nonce int, amount int) ([]entity.OutcomeMessage, error) {
	return uc.MessageRepository.FetchHistory(chatID, nonce, amount)
}

func (uc *DataBaseUseCase) GetLastNonce(chatID []byte) (int, error) {
	return uc.MessageRepository.GetLastNonce(chatID)
}
