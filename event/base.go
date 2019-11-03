package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/benjohns1/es-accounting/util/registry"
	"github.com/benjohns1/es-accounting/util/time"
	"github.com/benjohns1/es-accounting/util/uuid"
)

// Event transport header keys
const (
	HeaderEventID       = "Event-Id"
	HeaderEventType     = "Event-Type"
	HeaderAggregateID   = "Aggregate-Id"
	HeaderAggregateType = "Aggregate-Type"
)

type Event interface {
	Header() Header
}

type Raw struct {
	EventID       string        `json:"eventId"`
	EventType     string        `json:"eventType"`
	AggregateID   string        `json:"aggregateId"`
	AggregateType string        `json:"aggregateType"`
	Timestamp     time.JSONNano `json:"timestamp"`
	Data          string        `json:"data"`
}

func (e Raw) String() string {
	return fmt.Sprintf("EventID: %s\nEventType: %s\nAggregateID: %s\nAggregateType: %s\nTime: %s\nEvent: %s", e.EventID, e.EventType, e.AggregateID, e.AggregateType, e.Timestamp, e.Data)
}

type Header struct {
	AggregateID   string
	AggregateType string
}

func getTypeName(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func headers(e Event) (eventID, eventType, aggregateID, aggregateType string, err error) {
	eventID, err = uuid.New()
	if err != nil {
		return
	}

	eventType = getTypeName(e)
	if eventType == "" {
		err = fmt.Errorf("could not determine event type")
		return
	}

	h := e.Header()

	aggregateID = h.AggregateID
	if aggregateID == "" {
		err = fmt.Errorf("event header AggregateID must have a value")
		return
	}

	aggregateType = h.AggregateType
	if aggregateType == "" {
		err = fmt.Errorf("event header AggregateType must have a value")
	}
	return
}

// Publish publishes event to event store
func Publish(e Event) error {

	eventID, eventType, aggregateID, aggregateType, err := headers(e)
	if err != nil {
		return fmt.Errorf("error retrieving event header before publishing: %w (%v)", err, e)
	}

	eventJSON, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error encoding event: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:%s/event", registry.EventStoreHost, registry.EventStorePort), bytes.NewBuffer(eventJSON))
	if err != nil {
		return fmt.Errorf("error preparing event request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(HeaderEventID, eventID)
	req.Header.Add(HeaderEventType, eventType)
	req.Header.Add(HeaderAggregateID, aggregateID)
	req.Header.Add(HeaderAggregateType, aggregateType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending event: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		var msg string
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			msg = ": " + string(body)
		}
		return fmt.Errorf("error from event receiver: [%s]%s", resp.Status, msg)
	}
	return nil
}
