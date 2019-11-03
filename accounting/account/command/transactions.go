package main

import (
	"fmt"
	"sync"
	"time"
)

type Transaction struct {
	ID            string
	DebitAccount  string
	CreditAccount string
	Amount        int64
	Description   string
	Occurred      time.Time
}

var transactionMux = &sync.Mutex{}
var transactions = []Transaction{}

func addTransaction(t Transaction) {
	transactionMux.Lock()
	defer transactionMux.Unlock()
	transactions = append(transactions, t)
}

func deleteTransaction(tid string) error {
	transactionMux.Lock()
	defer transactionMux.Unlock()
	len := len(transactions)
	for i := len; i > 0; i-- {
		if transactions[i].ID == tid {
			copy(transactions[i:], transactions[i+1:])
			transactions = transactions[:len-1]
			return nil
		}
	}
	return fmt.Errorf("transaction %s not found to delete", tid)
}
