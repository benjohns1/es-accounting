package main

import (
	"accounting/event"
	"accounting/util/registry"
	timeutil "accounting/util/time"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	http.HandleFunc("/event", eventHandler)
	http.HandleFunc("/history", historyHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})
	log.Printf("listening on port %s", registry.EventStore)
	log.Fatal(http.ListenAndServe(":"+registry.EventStore, nil))
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
		return
	}
	addEventHandler(w, r)
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
		return
	}

	events := store

	aggregateType := r.URL.Query().Get("aggregateType")
	if aggregateType != "" {
		filtered := []event.Raw{}
		for _, e := range events {
			if e.AggregateType == aggregateType {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	// @TODO: filter on aggregate ID, timestamp, event type

	data, err := json.Marshal(events)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"unable to encode JSON response"}`))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
	return
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
var store = []event.Raw{}

func save(e event.Raw) error {
	storeMux.Lock()
	store = append(store, e)
	storeMux.Unlock()
	return nil
}

func addEventHandler(w http.ResponseWriter, r *http.Request) {

	// Read headers
	eventID := eventID(r.Header.Get(event.HeaderEventID))
	eventType := r.Header.Get(event.HeaderEventType)
	aggregateID := r.Header.Get(event.HeaderAggregateID)
	aggregateType := r.Header.Get(event.HeaderAggregateType)

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
	e := event.Raw{string(eventID), eventType, aggregateID, aggregateType, timeutil.JSONNano{time.Now()}, string(body)}
	err = save(e)
	if err != nil {
		writeLogResponse(w, http.StatusInternalServerError, "error saving event")
		return
	}
	log.Printf("saved event:\n%v\n", e)

	// Return success
	w.WriteHeader(http.StatusCreated)

	// Publish event to all others
	go func() {
		errs := broadcast(e, []string{
			fmt.Sprintf("http://localhost:%s/event", registry.AccountCommandEvent),
			fmt.Sprintf("http://localhost:%s/event", registry.AccountQueryEvent),
		})
		if len(errs) > 0 {
			strs := []string{}
			for _, err := range errs {
				strs = append(strs, err.Error())
			}
			log.Printf("errors broadcasting event: [%s]", strings.Join(strs, ", "))
		}
	}()
}

func writeLogResponse(w http.ResponseWriter, status int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Print(msg)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func addErr(errs []error, format string, a ...interface{}) {
	errs = append(errs, fmt.Errorf(format, a...))
}

func broadcast(e event.Raw, urls []string) (errs []error) {

	errs = make([]error, 0)

	client := &http.Client{}
	for _, url := range urls {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(e.Data)))
		if err != nil {
			addErr(errs, "error preparing event request: %w", err)
			continue
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add(event.HeaderEventID, string(e.EventID))
		req.Header.Add(event.HeaderEventType, e.EventType)
		req.Header.Add(event.HeaderAggregateID, e.AggregateID)
		req.Header.Add(event.HeaderAggregateType, e.AggregateType)

		resp, err := client.Do(req)
		if err != nil {
			addErr(errs, "error sending event: %w", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			var msg string
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				msg = ": " + string(body)
			}
			addErr(errs, "error from event receiver: [%s]%s", resp.Status, msg)
			continue
		}
	}
	return
}
