// +build !dockernetwork

// Package registry registers endpoints for dev-networking
package registry

// Registry endpoints
const (
	AccountCommandEventHost = "localhost"
	AccountCommandEventPort = "50101"
	AccountCommandAPIPort   = "7000"
	AccountQueryEventHost   = "localhost"
	AccountQueryEventPort   = "50102"
	AccountQueryAPIPort     = "7001"
	EventStoreHost          = "localhost"
	EventStorePort          = "50100"
)
