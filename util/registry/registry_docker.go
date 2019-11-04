// +build dockernetwork

// Package registry registers endpoints for dev-networking
package registry

// Registry endpoints
const (
	AccountCommandEventHost = "account-command"
	AccountCommandEventPort = "50101"
	AccountCommandAPIPort   = "7000"
	AccountQueryEventHost   = "account-query"
	AccountQueryEventPort   = "50102"
	AccountQueryAPIPort     = "7001"
	EventStoreHost          = "eventstore"
	EventStorePort          = "50100"
)
