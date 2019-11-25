package repo

import (
	"sync"

	"github.com/benjohns1/es-accounting/event"
)

// InMem in-memory event store
type InMem struct {
	storeMux *sync.Mutex
	index    int64
	store    []event.Raw
}

// NewInMem returns an in-memory event store repository
func NewInMem() *InMem {
	return &InMem{
		storeMux: &sync.Mutex{},
		store:    []event.Raw{},
	}
}

// Save saves an event to the store and returns it with a populated index
func (r *InMem) Save(e event.Raw) (event.Raw, error) {
	r.storeMux.Lock()
	defer r.storeMux.Unlock()
	r.index++
	e.EventIndex = r.index
	r.store = append(r.store, e)
	return e, nil
}

// GetEvents returns the slice of stored events
func (r InMem) GetEvents() []event.Raw {
	return r.store
}
