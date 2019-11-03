package event

import (
	"accounting/util/time"
)

type TransactionAdded struct {
	TransactionID string    `json:"transactionId"`
	DebitAccount  string    `json:"debitAccount,omitempty"`
	CreditAccount string    `json:"creditAccount,omitempty"`
	Amount        int64     `json:"amount"`
	Description   string    `json:"description,omitempty"`
	Occurred      time.JSON `json:"occurred"`
}

func (e TransactionAdded) Header() Header {
	return Header{
		AggregateID:   e.TransactionID,
		AggregateType: "Transaction",
	}
}
