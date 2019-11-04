package main

import timeutil "github.com/benjohns1/es-accounting/util/time"

type Query interface{}

type ListTransactions struct {
	RollbackTime timeutil.JSONNano `json:"rollbackTime"`
}

type GetAccountBalance struct {
	RollbackTime timeutil.JSONNano `json:"rollbackTime"`
}
