package entity

type IncomeMessage struct {
	Type    string `json:"type"`
	Message Message `json:"message"`
}

type Message struct {
	Nonce     int    `json:"nonce"`
	ChatID    string `json:"queueId"`
	Signature string `json:"signature"`
	Content   string `json:"content"`
	ContentIV string `json:"contentIV"`
}
