package main

import (
	"fmt"
	"sync"
	"time"
)

// Current aggregate state of transactions
var state = New()

type Transactions struct {
	mux          *sync.Mutex
	transactions []Transaction
	balance      Balance
}

func New() *Transactions {
	return &Transactions{
		mux:          &sync.Mutex{},
		transactions: []Transaction{},
		balance:      Balance{},
	}
}

func NewSnapshot(snapshot time.Time) *Transactions {
	snapshotState := New()
	snapshotState.loadState(snapshot)
	return snapshotState
}

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

func query(q Query) (interface{}, error) {
	// @TODO: when there is a Snapshot filter, cache state up to that point and retrieve it instead
	// Need to cache state with an explicit event version number (along with timestamps) and determine which cached state to load for a given timestamp
	switch q.(type) {
	case ListTransactions:
		query := q.(ListTransactions)
		var snapshotState *Transactions
		if query.Snapshot == nil {
			snapshotState = state
		} else {
			snapshotState = NewSnapshot(*query.Snapshot)
		}
		return snapshotState.transactions, nil
	case GetAccountBalance:
		query := q.(GetAccountBalance)
		var snapshotState *Transactions
		if query.Snapshot == nil {
			snapshotState = state
		} else {
			snapshotState = NewSnapshot(*query.Snapshot)
		}
		return snapshotState.balance, nil
	default:
		return nil, fmt.Errorf("unknown query type")
	}
}

func (t *Transactions) addTransaction(new Transaction) {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.transactions = append(t.transactions, new)
	t.balance.Balance += new.Amount
}

func (t *Transactions) deleteTransaction(tid string) error {
	t.mux.Lock()
	defer t.mux.Unlock()
	len := len(t.transactions)
	for i := len - 1; i >= 0; i-- {
		if t.transactions[i].ID == tid {
			amount := t.transactions[i].Amount
			copy(t.transactions[i:], t.transactions[i+1:])
			t.transactions = t.transactions[:len-1]
			t.balance.Balance -= amount
			return nil
		}
	}
	return fmt.Errorf("transaction %s not found to delete", tid)
}
