module accounting/eventstore

go 1.13

replace accounting/events => ../events

require (
	accounting/events v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.1.1
)
