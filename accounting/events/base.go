package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/google/uuid"
)

type Event interface {
	SetID(string) error
	GetType() string
	GetAggregateID() string
	GetAggregateType() string
}

type BaseEvent struct {
	EventID       string `json:"eventId"`
	EventType     string `json:"eventType"`
	AggregateID   string `json:"aggregateId"`
	AggregateType string `json:"aggregateType"`
}

func (e *BaseEvent) SetID(id string) error {
	e.EventID = id
	return nil
}

func (e *BaseEvent) GetType() string {
	return e.EventType
}

func (e *BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

func (e *BaseEvent) GetAggregateType() string {
	return e.AggregateType
}

func getTypeName(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func validate(e Event) (eventType, aggregateID, aggregateType string, err error) {
	eventType = e.GetType()
	if eventType == "" {
		err = fmt.Errorf("GetType() returned zero value")
		return
	}
	aggregateID = e.GetAggregateID()
	if aggregateID == "" {
		err = fmt.Errorf("GetAggregateID() returned zero value")
		return
	}
	aggregateType = e.GetAggregateType()
	if aggregateType == "" {
		err = fmt.Errorf("GetAggregateType() returned zero value")
	}
	return

}

func Publish(e Event) error {

	id, err := generateGUID()
	if err != nil {
		return err
	}

	err = e.SetID(id)
	if err != nil {
		return err
	}

	eventType, aggregateID, aggregateType, err := validate(e)
	if err != nil {
		return fmt.Errorf("error validating event for publish: %w (%v)", err, e)
	}

	eventJSON, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error encoding event: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8081/event", bytes.NewBuffer(eventJSON))
	if err != nil {
		return fmt.Errorf("error preparing event request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Event-Id", id)
	req.Header.Add("Event-Type", eventType)
	req.Header.Add("Aggregate-Id", aggregateID)
	req.Header.Add("Aggregate-Type", aggregateType)
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
		return fmt.Errorf("error storing event: [%s]%s", resp.Status, msg)
	}
	return nil
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
