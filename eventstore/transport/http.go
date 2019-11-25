package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/benjohns1/es-accounting/event"
	"github.com/benjohns1/es-accounting/eventstore"
	httputil "github.com/benjohns1/es-accounting/util/http"
	"github.com/benjohns1/es-accounting/util/registry"
	timeutil "github.com/benjohns1/es-accounting/util/time"
)

// HTTP transport handler
type HTTP struct {
	addEventFunc   func(e event.Raw) error
	getHistoryFunc func(filter eventstore.Filter) ([]event.Raw, error)
}

// NewHTTP creates an HTTP transport handler for the event store
func NewHTTP() *HTTP {
	return &HTTP{}
}

// SetAddEventFunc sets the addEventFunc to call when an event is received
func (t *HTTP) SetAddEventFunc(addEventFunc func(e event.Raw) error) {
	t.addEventFunc = addEventFunc
}

// SetGetHistoryFunc sets the getHistoryFunc to call when a history requests is received
func (t *HTTP) SetGetHistoryFunc(getHistoryFunc func(filter eventstore.Filter) ([]event.Raw, error)) {
	t.getHistoryFunc = getHistoryFunc
}

// Filter filters the getHistoryFunc events
type Filter struct {
	aggregateType string
}

// AggregateType returnes the aggregate type filter value
func (f Filter) AggregateType() string {
	return f.aggregateType
}

// Listen starts the HTTP server and listens on the registered port
func (t HTTP) Listen() error {

	http.HandleFunc("/event", t.addEventHandler)
	http.HandleFunc("/history", t.getHistoryHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid URI"}`))
	})
	log.Printf("listening on port %s", registry.EventStorePort)

	return http.ListenAndServe(":"+registry.EventStorePort, nil)
}

// Broadcast broadcasts the event to all subscribers
func (t HTTP) Broadcast(e event.Raw) {

	// Publish event to all others
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
}

func (t HTTP) addEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
		return
	}

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

	if err := t.addEventFunc(e); err != nil {
		httputil.WriteLogResponse(w, http.StatusInternalServerError, "error adding event")
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
}

func (t HTTP) getHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid HTTP method"}`))
		return
	}

	events, err := t.getHistoryFunc(Filter{
		aggregateType: r.URL.Query().Get("aggregateType"),
	})
	if err != nil {
		httputil.WriteErrStrJSONResponse(w, http.StatusInternalServerError, "error filtering events")
		return
	}

	data, err := json.Marshal(events)
	if err != nil {
		httputil.WriteErrStrJSONResponse(w, http.StatusBadRequest, "unable to encode JSON response")
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
	return
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
		if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
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

func addErr(errs *[]error, format string, a ...interface{}) {
	*errs = append(*errs, fmt.Errorf(format, a...))
}
