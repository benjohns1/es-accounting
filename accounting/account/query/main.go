package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/benjohns1/es-accounting/event"
	"github.com/benjohns1/es-accounting/util/registry"
)

func main() {

	ready := make(chan bool, 2)
	errCh := make(chan error)

	// Listen for events
	http.HandleFunc("/event", createEventListener(ready))
	go func() {
		log.Printf("event endpoint listening on port %s", registry.AccountQueryEventPort)
		errCh <- http.ListenAndServe(":"+registry.AccountQueryEventPort, nil)
	}()

	err := loadCurrentState()
	if err != nil {
		log.Fatalf("error loading current state: %v", err)
	}
	ready <- true

	// Listen for queries
	http.HandleFunc("/transaction", listTransactionsHandler)
	http.HandleFunc("/balance", getBalanceHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})

	go func() {
		log.Printf("query endpoints listening on port %s", registry.AccountQueryAPIPort)
		errCh <- http.ListenAndServe(":"+registry.AccountQueryAPIPort, nil)
	}()

	log.Fatal(<-errCh)
}

func loadCurrentState() error {
	client := &http.Client{}
	aggregateType := "Transaction"
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%s/history?aggregateType=%s", registry.EventStoreHost, registry.EventStorePort, aggregateType), bytes.NewBuffer([]byte{}))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response: %s %s", resp.Status, string(data))
	}
	events := []event.Raw{}
	err = json.Unmarshal(data, &events)
	if err != nil {
		return err
	}

	fmt.Printf("%d events received from eventstore\n", len(events))

	// Convert event to proper event type and replay to aggregate
	for _, raw := range events {
		err = replay(raw)
		if err != nil {
			return fmt.Errorf("halting replay: %w", err)
		}
	}

	return nil
}

func createEventListener(ready chan bool) func(w http.ResponseWriter, r *http.Request) {
	readyMux := &sync.Mutex{}
	aggregatesReady := false
	queueMux := &sync.Mutex{}
	replayQueue := []http.Request{}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid HTTP method"}`))
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

		// Read headers
		eventID := r.Header.Get(event.HeaderEventID)
		eventType := r.Header.Get(event.HeaderEventType)
		aggregateID := r.Header.Get(event.HeaderAggregateID)
		aggregateType := r.Header.Get(event.HeaderAggregateType)

		eventData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"couldn't read body"}`))
			return
		}
		err = apply(eventID, eventType, aggregateID, aggregateType, eventData)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"couldn't apply event"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}
}

func getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
	}

	processQuery("GetAccountBalance", GetAccountBalance{}, w)
}

func listTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
	}

	processQuery("ListTransactions", ListTransactions{}, w)
}

func processQuery(name string, q Query, w http.ResponseWriter) {
	// Query aggregate
	data, err := query(q)
	if err != nil {
		log.Printf("error running query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(data)
	if err != nil {
		log.Printf("error encoding response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("successfully processed %s query", name)
	w.Header().Add("Content-Type", "application/json")
	w.Write(jsonResponse)
}
