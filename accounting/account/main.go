package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"accounting/events"

	"github.com/google/uuid"
)

func main() {
	http.HandleFunc("/transaction", transactionHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})
	log.Printf("listening on port 8080")
	http.ListenAndServe(":8080", nil)
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

type addTransactionCommand struct {
	Amount int64 `json:"amount"`
}

type transaction struct {
	ID     string
	Amount int64
}

var transactions = map[string]transaction{}

func addTransactionHandler(w http.ResponseWriter, r *http.Request) {
	// Read command
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	atc := addTransactionCommand{}
	err = json.Unmarshal(body, &atc)
	if err != nil {
		log.Printf("error reading json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Process command
	tid, err := generateGUID()
	if err != nil {
		log.Printf("%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update aggregate
	t := transaction{
		ID:     tid,
		Amount: atc.Amount,
	}
	transactions[tid] = t

	// Publish event
	err = events.Publish(&events.TransactionAdded{
		BaseEvent: &events.BaseEvent{
			AggregateID:   tid,
			AggregateType: "transaction",
			EventType:     "TransactionAdded",
		},
		TransactionID: t.ID,
		Amount:        t.Amount,
	})
	if err != nil {
		// Error publishing event: undo command processing
		delete(transactions, tid)
		log.Printf("error publishing event: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("successfully processed command")
	w.Write([]byte(fmt.Sprintf(`{"transaction":{"id":"%s"}"}`, tid)))
}

func generateGUID() (string, error) {
	guids, err := generateGUIDs(1)
	return guids[0], err
}

func generateGUIDs(count int) ([]string, error) {
	uuids := []string{}
	for i := 0; i < count; i++ {
		id, err := uuid.NewUUID()
		idStr := id.String()
		if idStr == "" {
			err = fmt.Errorf("unable to create UUID")
		}
		if err != nil {
			return uuids, err
		}
		uuids = append(uuids, idStr)
	}
	return uuids, nil
}
