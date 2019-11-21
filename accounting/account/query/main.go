package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/benjohns1/es-accounting/event"
	httputil "github.com/benjohns1/es-accounting/util/http"
	"github.com/benjohns1/es-accounting/util/registry"
	timeutil "github.com/benjohns1/es-accounting/util/time"
)

func main() {

	ready := make(chan bool, 2)
	errCh := make(chan error)

	eventQueue := make(chan event.Raw, 100)

	// Listen for events
	http.HandleFunc("/event", createEventListener(eventQueue, ready))
	go func() {
		log.Printf("event endpoint listening on port %s", registry.AccountQueryEventPort)
		errCh <- http.ListenAndServe(":"+registry.AccountQueryEventPort, nil)
	}()

	err := event.LoadState("Transaction", func(raw event.Raw) error {
		addEvent(raw)
		return state.replayEvent(raw)
	})
	if err != nil {
		log.Fatalf("error loading current state: %v", err)
	}
	ready <- true

	// Listen for queries
	http.HandleFunc("/transaction", listTransactionsHandler)
	http.HandleFunc("/balance", getBalanceHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid URI")
	})

	go func() {
		log.Printf("query endpoints listening on port %s", registry.AccountQueryAPIPort)
		errCh <- http.ListenAndServe(":"+registry.AccountQueryAPIPort, nil)
	}()

	for {
		select {
		case raw := <-eventQueue:
			addEvent(raw)
			err := state.applyEvent(raw)
			if err != nil {
				log.Printf("error handling event: %v", err)
			}
		case err := <-errCh:
			log.Fatal(err)
		}
	}
}

func createEventListener(eventQueue chan event.Raw, ready chan bool) func(w http.ResponseWriter, r *http.Request) {
	readyMux := &sync.Mutex{}
	aggregatesReady := false
	queueMux := &sync.Mutex{}
	replayQueue := []http.Request{}

	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid HTTP method")
			return
		}

		if !aggregatesReady {
			select {
			case <-ready:
				readyMux.Lock()
				// @TODO: here we need to process the queue of events (and ignore duplicates)
				// once we're done, there could be MORE events added to the queue, so we need to keep doing this until there are no events in the queue
				// OR we migrate immediately to RabbitMQ and just let the events stack up there until we're ready
				if len(replayQueue) > 0 {
					log.Fatalf("%d unprocessed events in the queue!", len(replayQueue))
				}
				aggregatesReady = true
				readyMux.Unlock()
			default:
				log.Print("received new event, but the aggregate store hasn't finished loading, queueing event request")
				queueMux.Lock()
				defer queueMux.Unlock()
				replayQueue = append(replayQueue, *r)
				return
			}
		}

		log.Printf("event received")

		var timestamp timeutil.JSONNano
		if parsed, err := time.Parse(time.RFC3339Nano, r.Header.Get(event.HeaderTimestamp)); err != nil {
			log.Printf("error: event timestamp could not be parsed: %v", err)
		} else {
			timestamp = timeutil.JSONNano{Time: parsed}
		}

		// Read headers
		eIdx, err := strconv.ParseInt(r.Header.Get(event.HeaderEventIndex), 10, 64)
		if err != nil {
			log.Printf("error: couldn't parse event index: %v", err)
		}
		raw := event.Raw{
			EventIndex:    eIdx,
			EventID:       r.Header.Get(event.HeaderEventID),
			EventType:     r.Header.Get(event.HeaderEventType),
			AggregateID:   r.Header.Get(event.HeaderAggregateID),
			AggregateType: r.Header.Get(event.HeaderAggregateType),
			Timestamp:     timestamp,
		}

		eventData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "couldn't read body")
			return
		}
		raw.Data = string(eventData)
		eventQueue <- raw
		w.WriteHeader(http.StatusAccepted)
		return
	}
}

func getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid HTTP method")
		return
	}

	processQuery("GetAccountBalance", GetAccountBalance{Snapshot: parseSnapshotParam(w, r)}, w)
}

func listTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid HTTP method")
		return
	}

	processQuery("ListTransactions", ListTransactions{Snapshot: parseSnapshotParam(w, r)}, w)
}

func parseSnapshotParam(w http.ResponseWriter, r *http.Request) *time.Time {
	var snapshot *time.Time
	if vals, ok := r.URL.Query()["snapshot"]; ok && len(vals) > 0 {
		val, err := time.Parse(time.RFC3339Nano, vals[0])
		if err != nil {
			httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid snapshot parameter")
			return nil
		}
		snapshot = &val
		log.Printf("Snapshot: %v", snapshot)
	}
	return snapshot
}

func processQuery(name string, q Query, w http.ResponseWriter) {

	// Query aggregate state
	data, err := query(q)
	if err != nil {
		httputil.WriteErrJSONResponse(w, http.StatusInternalServerError, fmt.Errorf("error running query: %w", err))
		return
	}

	jsonResponse, err := json.Marshal(data)
	if err != nil {
		httputil.WriteErrJSONResponse(w, http.StatusInternalServerError, fmt.Errorf("error encoding response: %w", err))
		return
	}

	log.Printf("successfully processed %s query", name)
	w.Header().Add("Content-Type", "application/json")
	w.Write(jsonResponse)
}
