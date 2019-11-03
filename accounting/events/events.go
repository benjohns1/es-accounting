package events

type TransactionAdded struct {
	*BaseEvent
	TransactionID string `json:"transactionId"`
	Amount        int64  `json:"amount"`
}
