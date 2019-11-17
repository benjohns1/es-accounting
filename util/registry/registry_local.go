// +build !dockernetwork

// Package registry registers endpoints for dev-networking
package registry

// Registry endpoints
const (
	AccountCommandEventHost = "localhost"
	AccountCommandEventPort = "9000"
	AccountCommandAPIPort   = "7000"
	AccountQueryEventHost   = "localhost"
	AccountQueryEventPort   = "9001"
	AccountQueryAPIPort     = "7001"
	EventStoreHost          = "localhost"
	EventStorePort          = "8000"
)
