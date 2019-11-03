package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

func main() {
	http.HandleFunc("/event", eventHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})
	log.Printf("listening on port 8081")
	http.ListenAndServe(":8081", nil)
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		addEventHandler(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
	}
}

type addTransactionCommand struct {
	Amount int64 `json:"amount"`
}

type transaction struct {
	ID     string
	Amount int64
}

type eventID string

var storeMux = &sync.Mutex{}
var store = make(map[eventID]event)

func save(e event) error {
	storeMux.Lock()
	store[e.eventID] = e
	storeMux.Unlock()
	return nil
}

type event struct {
	eventID       eventID
	eventType     string
	aggregateID   string
	aggregateType string
	data          []byte
}

func (e event) String() string {
	return fmt.Sprintf(`ID: %s, Type: %s, AggregateID: %s, AggregateType: %s, Data: %s`, e.eventID, e.eventType, e.aggregateID, e.aggregateType, string(e.data))
}

func addEventHandler(w http.ResponseWriter, r *http.Request) {

	// Read headers
	eventID := eventID(r.Header.Get("Event-Id"))
	eventType := r.Header.Get("Event-Type")
	aggregateID := r.Header.Get("Aggregate-Id")
	aggregateType := r.Header.Get("Aggregate-Type")

	// Verify headers
	if eventID == "" || eventType == "" || aggregateID == "" || aggregateType == "" {
		writeLogResponse(w, http.StatusBadRequest, "required header missing for event: %s", eventID)
		return
	}

	// Read event body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeLogResponse(w, http.StatusInternalServerError, "error reading request body")
		return
	}

	// Save event
	e := event{eventID, eventType, aggregateID, aggregateType, body}
	err = save(e)
	if err != nil {
		writeLogResponse(w, http.StatusInternalServerError, "error saving event")
		return
	}
	log.Printf("saved event: %v", e)

	// Publish event to all others
	// @TODO

	w.WriteHeader(http.StatusCreated)
}

func writeLogResponse(w http.ResponseWriter, status int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Print(msg)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}
