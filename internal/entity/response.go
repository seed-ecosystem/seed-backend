package entity

type NewEventResponse struct {
	Type  string         `json:"type"`
	Event NewEventDetail `json:"event"`
}

type NewEventDetail struct {
	Type    string         `json:"type"`
	Message Message `json:"message"`
}

type WaitEventResponse struct {
	Type  string          `json:"type"`
	Event WaitEventDetail `json:"event"`
}

type WaitEventDetail struct {
	Type   string `json:"type"`
	ChatID string `json:"queueId"`
}

type StatusResponse struct {
	Type   string `json:"type"`
	Status bool   `json:"status"`
}
