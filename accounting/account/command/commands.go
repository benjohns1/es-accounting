package main

import timeutil "github.com/benjohns1/es-accounting/util/time"

type Command interface{}

type AddTransactionCommand struct {
	DebitAccount  string            `json:"debitAccount"`
	CreditAccount string            `json:"creditAccount"`
	Amount        int64             `json:"amount"`
	Description   string            `json:"description"`
	Occurred      timeutil.JSONNano `json:"occurred"`
}

type DeleteTransactionCommand struct {
	TransactionID string `json:"id"`
}
