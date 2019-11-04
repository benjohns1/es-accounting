package event

import (
	"github.com/benjohns1/es-accounting/util/time"
)

type TransactionAdded struct {
	TransactionID string        `json:"transactionId"`
	DebitAccount  string        `json:"debitAccount,omitempty"`
	CreditAccount string        `json:"creditAccount,omitempty"`
	Amount        int64         `json:"amount"`
	Description   string        `json:"description,omitempty"`
	Occurred      time.JSONNano `json:"occurred"`
}

func (e TransactionAdded) Header() Header {
	return Header{
		AggregateID:   e.TransactionID,
		AggregateType: "Transaction",
	}
}

type TransactionDeleted struct {
	TransactionID string `json:"transactionId"`
}

func (e TransactionDeleted) Header() Header {
	return Header{
		AggregateID:   e.TransactionID,
		AggregateType: "Transaction",
	}
}
