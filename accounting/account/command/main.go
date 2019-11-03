package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/benjohns1/es-accounting/event"
	"github.com/benjohns1/es-accounting/util/registry"
	timeutil "github.com/benjohns1/es-accounting/util/time"
	"github.com/benjohns1/es-accounting/util/uuid"
)

func main() {

	ready := make(chan bool, 2)
	errCh := make(chan error)

	// Listen for events
	http.HandleFunc("/event", createEventListener(ready))
	go func() {
		log.Printf("event endpoint listening on port %s", registry.AccountCommandEventPort)
		errCh <- http.ListenAndServe(":"+registry.AccountCommandEventPort, nil)
	}()

	// @TODO: load state from event store
	ready <- true

	// Listen for commands
	http.HandleFunc("/transaction", transactionHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})
	go func() {
		log.Printf("command endpoints listening on port %s", registry.AccountCommandAPIPort)
		errCh <- http.ListenAndServe(":"+registry.AccountCommandAPIPort, nil)
	}()

	log.Fatal(<-errCh)
}

func createEventListener(ready chan bool) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid HTTP method"}`))
		}

		log.Printf("TODO: handle inbound events")
	}
}

func transactionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		addTransactionHandler(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
	}
}

func addTransactionHandler(w http.ResponseWriter, r *http.Request) {
	// Read command
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	atc := AddTransactionCommand{}
	err = json.Unmarshal(body, &atc)
	if err != nil {
		log.Printf("error reading json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Process command
	tid, err := uuid.New()
	if err != nil {
		log.Printf("%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update aggregate
	t := Transaction{
		ID:            tid,
		DebitAccount:  atc.DebitAccount,
		CreditAccount: atc.CreditAccount,
		Amount:        atc.Amount,
		Description:   atc.Description,
		Occurred:      atc.Occurred.Time,
	}
	addTransaction(t)

	// Publish event
	err = event.Publish(&event.TransactionAdded{
		TransactionID: t.ID,
		DebitAccount:  t.DebitAccount,
		CreditAccount: t.CreditAccount,
		Amount:        t.Amount,
		Description:   t.Description,
		Occurred:      timeutil.JSON{Time: t.Occurred},
	})
	if err != nil {
		// Error publishing event: undo command processing
		deleteTransaction(tid)
		log.Printf("error publishing event: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("successfully processed AddTransaction command")
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"id":"%s"}`, tid)))
}
