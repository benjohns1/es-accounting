package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/benjohns1/es-accounting/event"
)

type Transaction struct {
	ID            string
	DebitAccount  string
	CreditAccount string
	Amount        int64
	Description   string
	Occurred      time.Time
}

type Balance struct {
	Balance int64
}

var transactionMux = &sync.Mutex{}
var transactions = []Transaction{}
var balance Balance = Balance{}

func query(q Query) (interface{}, error) {
	switch q.(type) {
	case ListTransactions:
		return transactions, nil
	case GetAccountBalance:
		return balance, nil
	default:
		return nil, fmt.Errorf("unknown query type")
	}
}

func replay(raw event.Raw) error {
	switch raw.EventType {
	case "TransactionAdded":
		applyTransactionAdded([]byte(raw.Data))
	default:
		return fmt.Errorf("unknown event type")
	}

	return nil
}

func apply(eventID, eventType, aggregateID, aggregateType string, eventData []byte) error {
	switch eventType {
	case "TransactionAdded":
		err := applyTransactionAdded(eventData)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyTransactionAdded(data []byte) error {
	e := event.TransactionAdded{}
	err := json.Unmarshal(data, &e)
	if err != nil {
		return fmt.Errorf("error decoding event data: %v", string(data))
	}

	// Update aggregate
	t := Transaction{
		ID:            e.TransactionID,
		DebitAccount:  e.DebitAccount,
		CreditAccount: e.CreditAccount,
		Amount:        e.Amount,
		Description:   e.Description,
		Occurred:      e.Occurred.Time,
	}
	addTransaction(t)
	return nil
}

func addTransaction(t Transaction) {
	transactionMux.Lock()
	defer transactionMux.Unlock()
	balance.Balance += t.Amount
	transactions = append(transactions, t)
}
