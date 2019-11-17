package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/benjohns1/es-accounting/util/registry"
)

// LoadState loads an aggregate's current state from the event store
func LoadState(aggregateType string, replayFunc func(event Raw) error) error {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%s/history?aggregateType=%s", registry.EventStoreHost, registry.EventStorePort, aggregateType), bytes.NewBuffer([]byte{}))
	if err != nil {
		return err
	}

	const maxAttempts = 20
	const retrySleep = 3
	var resp *http.Response
	for i := 1; i <= maxAttempts; i++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		if i >= maxAttempts {
			return fmt.Errorf("failed connecting after %d attempts: %v", i, err)
		}
		time.Sleep(retrySleep * time.Second)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response: %s %s", resp.Status, string(data))
	}
	events := []Raw{}
	err = json.Unmarshal(data, &events)
	if err != nil {
		return err
	}

	fmt.Printf("%d events received from eventstore\n", len(events))

	// Convert event to proper event type and replay to aggregate
	for _, raw := range events {
		err = replayFunc(raw)
		if err != nil {
			return fmt.Errorf("halting replay: %w", err)
		}
	}

	return nil
}
