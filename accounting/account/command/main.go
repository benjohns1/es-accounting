package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/benjohns1/es-accounting/event"
	httputil "github.com/benjohns1/es-accounting/util/http"
	"github.com/benjohns1/es-accounting/util/registry"
	timeutil "github.com/benjohns1/es-accounting/util/time"
	"github.com/benjohns1/es-accounting/util/uuid"
	guid "github.com/google/uuid"
)

func main() {

	ready := make(chan bool, 2)
	errCh := make(chan error)

	eventQueue := make(chan event.Raw, 100)

	// Listen for events
	http.HandleFunc("/event", createEventListener(eventQueue, ready))
	go func() {
		log.Printf("event endpoint listening on port %s", registry.AccountCommandEventPort)
		errCh <- http.ListenAndServe(":"+registry.AccountCommandEventPort, nil)
	}()

	err := event.LoadState("Transaction", replayEvent)
	if err != nil {
		log.Fatalf("error loading current state: %v", err)
	}
	ready <- true

	commandQueue := make(chan Command, 100)

	// Listen for commands
	http.HandleFunc("/transaction", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid HTTP method")
			return
		}
		addTransactionHandler(commandQueue, w, r)
	})
	http.HandleFunc("/transaction/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if r.Method != http.MethodDelete {
			httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid HTTP method")
			return
		}
		deleteTransactionHandler(commandQueue, w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid URI")
	})
	go func() {
		log.Printf("command endpoints listening on port %s", registry.AccountCommandAPIPort)
		errCh <- http.ListenAndServe(":"+registry.AccountCommandAPIPort, nil)
	}()

	for {
		select {
		case command := <-commandQueue:
			err := handleCommand(command)
			if err != nil {
				log.Printf("error handling command: %v", err)
			}
		case raw := <-eventQueue:
			err := applyEvent(raw)
			if err != nil {
				log.Printf("error handling event: %v", err)
			}
		case err := <-errCh:
			log.Fatal(err)
		}
	}
}

func handleCommand(c Command) error {
	switch c.(type) {
	case AddTransactionCommand:
		// Process command
		atc := c.(AddTransactionCommand)
		tid, err := uuid.New()
		if err != nil {
			return err
		}
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
			Occurred:      timeutil.JSONNano{Time: t.Occurred},
		})
		if err != nil {
			// Error publishing event: undo command processing
			_, deleteErr := deleteTransaction(tid)
			if deleteErr != nil {
				log.Printf("fatal error trying to undo AddTransaction command: %v", deleteErr)
				os.Exit(1) // how to recover?
			}
			return fmt.Errorf("error publishing event: %w", err)
		}

		log.Printf("successfully processed AddTransaction command")

	case DeleteTransactionCommand:
		// Process command
		dtc := c.(DeleteTransactionCommand)
		deletedTransaction, err := deleteTransaction(dtc.TransactionID)
		if err != nil {
			return fmt.Errorf("error deleting transaction: %w", err)
		}

		// Publish event
		err = event.Publish(&event.TransactionDeleted{
			TransactionID: dtc.TransactionID,
		})
		if err != nil {
			// Error publishing event: undo command processing
			addTransaction(deletedTransaction)
			return fmt.Errorf("error publishing event: %w", err)
		}

		log.Printf("successfully processed DeleteTransaction command")

	default:
		return fmt.Errorf("unknown command: %v", c)
	}
	return nil
}

func createEventListener(eventQueue chan event.Raw, ready chan bool) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid HTTP method")
			return
		}

		log.Printf("TODO: parse inbound events, send on eventQueue back to main goroutine")
	}
}

func deleteTransactionHandler(commandQueue chan Command, w http.ResponseWriter, r *http.Request) {
	// Read command
	pieces := strings.Split(r.URL.Path, "/")
	if len(pieces) != 3 {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "invalid path")
		return
	}
	transactionGUID, err := guid.Parse(pieces[2])
	if err != nil {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "couldn't parse transaction ID")
		return
	}
	tid := transactionGUID.String()
	dtc := DeleteTransactionCommand{
		TransactionID: tid,
	}

	commandQueue <- dtc

	httputil.WriteLogResponse(w, http.StatusAccepted, `{"commandType":"DeleteTransactionCommand"}`)
}

func addTransactionHandler(commandQueue chan Command, w http.ResponseWriter, r *http.Request) {
	// Read command
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httputil.WriteErrJSONResponse(w, http.StatusInternalServerError, fmt.Errorf("error reading body: %w", err))
		return
	}

	atc := AddTransactionCommand{}
	err = json.Unmarshal(body, &atc)
	if err != nil {
		httputil.WriteErrJSONResponse(w, http.StatusInternalServerError, fmt.Errorf("error reading json: %w", err))
		return
	}

	commandQueue <- atc

	httputil.WriteLogResponse(w, http.StatusAccepted, `{"commandType":"AddTransactionCommand"}`)
}
