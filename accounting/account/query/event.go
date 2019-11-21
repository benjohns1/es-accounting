package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/benjohns1/es-accounting/event"
)

var eventsMux = &sync.Mutex{}
var events = []event.Raw{}

func addEvent(raw event.Raw) {
	eventsMux.Lock()
	events = append(events, raw)
	eventsMux.Unlock()
}

func (t *Transactions) loadState(snapshot time.Time) error {
	eventsMux.Lock()
	defer eventsMux.Unlock()
	for _, raw := range events {
		if raw.Timestamp.Time.After(snapshot) {
			return nil
		}
		err := t.replayEvent(raw)
		if err != nil {
			return fmt.Errorf("halting replay: %w", err)
		}
	}
	return nil
}

func (t *Transactions) replayEvent(raw event.Raw) error {
	log.Printf("replaying event:\n%v", raw)
	switch raw.EventType {
	case "TransactionAdded":
		return t.applyTransactionAdded(raw.Data)
	case "TransactionDeleted":
		return t.applyTransactionDeleted(raw.Data)
	default:
		log.Printf("unknown event type: %s", raw.EventType)
	}

	return nil
}

func (t *Transactions) applyEvent(raw event.Raw) error {
	log.Printf("applying event:\n%v", raw)
	switch raw.EventType {
	case "TransactionAdded":
		return t.applyTransactionAdded(raw.Data)
	case "TransactionDeleted":
		return t.applyTransactionDeleted(raw.Data)
	}
	return nil
}

func (t *Transactions) applyTransactionAdded(data string) error {
	e := event.TransactionAdded{}
	err := json.Unmarshal([]byte(data), &e)
	if err != nil {
		return fmt.Errorf("error decoding event data: %v", data)
	}

	// Update aggregate
	transaction := Transaction{
		ID:            e.TransactionID,
		DebitAccount:  e.DebitAccount,
		CreditAccount: e.CreditAccount,
		Amount:        e.Amount,
		Description:   e.Description,
		Occurred:      e.Occurred.Time,
	}
	t.addTransaction(transaction)
	return nil
}

func (t *Transactions) applyTransactionDeleted(data string) error {
	e := event.TransactionDeleted{}
	err := json.Unmarshal([]byte(data), &e)
	if err != nil {
		return fmt.Errorf("error decoding event data: %v", data)
	}

	return t.deleteTransaction(e.TransactionID)
}
