package main

import timeutil "accounting/util/time"

type AddTransactionCommand struct {
	DebitAccount  string            `json:"debitAccount"`
	CreditAccount string            `json:"creditAccount"`
	Amount        int64             `json:"amount"`
	Description   string            `json:"description"`
	Occurred      timeutil.JSONUnix `json:"occurred"`
}
