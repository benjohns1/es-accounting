package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benjohns1/es-accounting/event"
	httputil "github.com/benjohns1/es-accounting/util/http"
	"github.com/benjohns1/es-accounting/util/registry"
	timeutil "github.com/benjohns1/es-accounting/util/time"
)

func main() {
	http.HandleFunc("/event", eventHandler)
	http.HandleFunc("/history", historyHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})
	log.Printf("listening on port %s", registry.EventStorePort)
	log.Fatal(http.ListenAndServe(":"+registry.EventStorePort, nil))
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
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "unable to encode JSON response")
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

var storeMux = &sync.Mutex{}
var index int64
var store = []event.Raw{}

func save(e event.Raw) (event.Raw, error) {
	storeMux.Lock()
	index++
	e.EventIndex = index
	store = append(store, e)
	storeMux.Unlock()
	return e, nil
}

func addEventHandler(w http.ResponseWriter, r *http.Request) {

	// Read headers
	eventID := r.Header.Get(event.HeaderEventID)
	eventType := r.Header.Get(event.HeaderEventType)
	aggregateID := r.Header.Get(event.HeaderAggregateID)
	aggregateType := r.Header.Get(event.HeaderAggregateType)

	// Verify headers
	if eventID == "" || eventType == "" || aggregateID == "" || aggregateType == "" {
		httputil.WriteLogResponse(w, http.StatusBadRequest, "required header missing for event: %s", eventID)
		return
	}

	// Read event body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httputil.WriteLogResponse(w, http.StatusInternalServerError, "error reading request body")
		return
	}

	// Save event
	e := event.Raw{
		EventID:       eventID,
		EventType:     eventType,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Timestamp:     timeutil.JSONNano{Time: time.Now()},
		Data:          string(body),
	}
	e, err = save(e)
	if err != nil {
		httputil.WriteLogResponse(w, http.StatusInternalServerError, "error saving event")
		return
	}
	log.Printf("saved event:\n%v\n", e)

	// Return success
	w.WriteHeader(http.StatusCreated)

	// Publish event to all others
	go func() {
		errs := broadcast(e, []string{
			fmt.Sprintf("http://%s:%s/event", registry.AccountCommandEventHost, registry.AccountCommandEventPort),
			fmt.Sprintf("http://%s:%s/event", registry.AccountQueryEventHost, registry.AccountQueryEventPort),
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

func addErr(errs *[]error, format string, a ...interface{}) {
	*errs = append(*errs, fmt.Errorf(format, a...))
}

func broadcast(e event.Raw, urls []string) (errs []error) {

	errs = make([]error, 0)

	client := &http.Client{}
	for _, url := range urls {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(e.Data)))
		if err != nil {
			addErr(&errs, "error preparing event request: %w", err)
			continue
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add(event.HeaderEventIndex, strconv.FormatInt(e.EventIndex, 10))
		req.Header.Add(event.HeaderEventID, e.EventID)
		req.Header.Add(event.HeaderEventType, e.EventType)
		req.Header.Add(event.HeaderAggregateID, e.AggregateID)
		req.Header.Add(event.HeaderAggregateType, e.AggregateType)
		req.Header.Add(event.HeaderTimestamp, e.Timestamp.String())

		resp, err := client.Do(req)
		if err != nil {
			addErr(&errs, "error sending event: %w", err)
			continue
		}
		if resp.StatusCode != http.StatusAccepted || resp.StatusCode != http.StatusOK {
			var msg string
			if body, err := ioutil.ReadAll(resp.Body); err == nil && len(body) > 0 {
				msg = ": " + string(body)
			}
			addErr(&errs, "error from event receiver: [%s]%s", resp.Status, msg)
			continue
		}
	}
	return
}
