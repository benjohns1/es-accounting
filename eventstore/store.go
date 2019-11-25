package eventstore

import (
	"log"

	"github.com/benjohns1/es-accounting/event"
)

// Repo specifies the event store repository interface
type Repo interface {
	Save(e event.Raw) (event.Raw, error)
	GetEvents() []event.Raw
}

// Transport specifies the event store transport interface
type Transport interface {
	SetAddEventFunc(func(e event.Raw) error)
	SetGetHistoryFunc(func(filter Filter) ([]event.Raw, error))
	Listen() error
	Broadcast(e event.Raw)
}

// Filter specifies which filters can be passed to getHistoryFunc
type Filter interface {
	AggregateType() string
}

// EventStore wraps the event store dependencies
type EventStore struct {
	Repo      Repo
	Transport Transport
}

// Start starts listening on the transport for events
func (es EventStore) Start() {
	es.Transport.SetAddEventFunc(es.addEvent)
	es.Transport.SetGetHistoryFunc(es.getHistory)

	errs := make(chan error)
	done := make(chan bool)
	go func() {
		if err := es.Transport.Listen(); err != nil {
			errs <- err
			return
		}
		done <- true
	}()

	select {
	case err := <-errs:
		log.Fatalf("error listening: %v", err)
	case <-done:
		log.Print("stopped listening")
	}
}

func (es EventStore) addEvent(e event.Raw) error {
	e, err := es.Repo.Save(e)
	if err != nil {
		return err
	}
	log.Printf("saved event:\n%v\n", e)

	go es.Transport.Broadcast(e)

	return nil
}

func (es EventStore) getHistory(filter Filter) ([]event.Raw, error) {

	events := es.Repo.GetEvents()

	if filter == nil {
		return events, nil
	}

	filtered := []event.Raw{}
	for _, e := range events {
		if failsStringFilter(filter.AggregateType(), e.AggregateType) {
			continue
		}

		// @TODO: filter on aggregate ID, timestamp, event type, etc

		filtered = append(filtered, e)
	}

	return filtered, nil
}

func failsStringFilter(filter, value string) bool {
	return filter != "" && value != filter
}
