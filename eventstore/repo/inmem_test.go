package repo

import (
	"testing"

	"github.com/benjohns1/es-accounting/event"
	timeutil "github.com/benjohns1/es-accounting/util/time"
)

func TestInMem_Save(t *testing.T) {
	type args struct {
		e event.Raw
	}
	tests := []struct {
		name      string
		r         *InMem
		args      args
		wantIndex int64
		wantErr   bool
	}{
		{
			name: "should add an event with index 1",
			r:    NewInMem(),
			args: args{e: event.Raw{
				EventID:       "test-id",
				EventType:     "test-type",
				AggregateID:   "test-aggregate",
				AggregateType: "test-aggregate-type",
				Timestamp:     timeutil.JSONNano{},
				Data:          "test-event-data",
			}},
			wantIndex: 1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.Save(tt.args.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("InMem.Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.EventIndex != tt.wantIndex {
				t.Errorf("InMem.Save() EventIndex = %v, wantIndex %v", got.EventIndex, tt.wantIndex)
			}
		})
	}
}

func TestInMem_GetEvents(t *testing.T) {
	tests := []struct {
		name        string
		r           *InMem
		wantCount   int
		wantEventID string
	}{
		{
			name: "should get event that was just saved",
			r: func() *InMem {
				r := NewInMem()
				_, err := r.Save(event.Raw{
					EventID:       "test-id-12345",
					EventType:     "test-type",
					AggregateID:   "test-aggregate",
					AggregateType: "test-aggregate-type",
					Timestamp:     timeutil.JSONNano{},
					Data:          "test-event-data",
				})
				if err != nil {
					t.Fatalf("could not save event")
				}
				return r
			}(),
			wantCount:   1,
			wantEventID: "test-id-12345",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := tt.r.GetEvents()
			if len(events) != 1 {
				t.Errorf("InMem.GetEvents() count = %v, wantCount %v", len(events), tt.wantCount)
				return
			}
			if events[0].EventID != tt.wantEventID {
				t.Errorf("InMem.GetEvents() EventID = %v, wantEventID %v", events[0].EventID, tt.wantEventID)
			}
		})
	}
}
