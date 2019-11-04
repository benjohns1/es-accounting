package main

import (
	"encoding/json"
	"fmt"
	"log"
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

var transactions = []Transaction{}

func replayEvent(raw event.Raw) error {
	switch raw.EventType {
	case "TransactionAdded":
		return applyTransactionAdded([]byte(raw.Data))
	case "TransactionDeleted":
		return applyTransactionDeleted([]byte(raw.Data))
	default:
		log.Printf("unknown event type: %s", raw.EventType)
	}

	return nil
}

func applyEvent(raw event.Raw) error {
	log.Printf("TODO: HANDLE INCOMING EVENTS")
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

func applyTransactionDeleted(data []byte) error {
	e := event.TransactionDeleted{}
	err := json.Unmarshal(data, &e)
	if err != nil {
		return fmt.Errorf("error decoding event data: %v", string(data))
	}

	_, err = deleteTransaction(e.TransactionID)
	return err
}

func addTransaction(t Transaction) {
	transactions = append(transactions, t)
}

func deleteTransaction(tid string) (Transaction, error) {
	len := len(transactions)
	var deletedTransaction Transaction
	for i := len - 1; i >= 0; i-- {
		if transactions[i].ID == tid {
			deletedTransaction = Transaction(transactions[i])
			copy(transactions[i:], transactions[i+1:])
			transactions = transactions[:len-1]
			return deletedTransaction, nil
		}
	}
	return deletedTransaction, fmt.Errorf("transaction %s not found to delete", tid)
}
