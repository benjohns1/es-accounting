package main

import "time"

type Query interface{}

type ListTransactions struct {
	Snapshot *time.Time
}

type GetAccountBalance struct {
	Snapshot *time.Time
}
