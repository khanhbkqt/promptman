package ws

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent_SetsTimestamp(t *testing.T) {
	before := time.Now().UTC()
	event := NewEvent(EventDataChanged, DataChangedPayload{Source: "collections"})
	after := time.Now().UTC()

	if event.TS.Before(before) || event.TS.After(after) {
		t.Errorf("event.TS = %v, want between %v and %v", event.TS, before, after)
	}
}

func TestNewEvent_SetsType(t *testing.T) {
	event := NewEvent(EventStressTick, nil)
	if event.Type != EventStressTick {
		t.Errorf("event.Type = %q, want %q", event.Type, EventStressTick)
	}
}

func TestNewEvent_SetsPayload(t *testing.T) {
	payload := RequestCompletedPayload{ReqID: "req-1", Status: 200, Time: 42}
	event := NewEvent(EventRequestCompleted, payload)

	got, ok := event.Payload.(RequestCompletedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want RequestCompletedPayload", event.Payload)
	}
	if got.ReqID != "req-1" || got.Status != 200 || got.Time != 42 {
		t.Errorf("payload = %+v, want {req-1, 200, 42}", got)
	}
}

func TestEvent_JSONSerialization(t *testing.T) {
	event := NewEvent(EventTestCompleted, TestCompletedPayload{
		Collection: "users",
		Passed:     9,
		Failed:     1,
		Total:      10,
		Duration:   1500,
	})

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded["type"] != EventTestCompleted {
		t.Errorf("type = %v, want %v", decoded["type"], EventTestCompleted)
	}

	payload, ok := decoded["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload type = %T, want map", decoded["payload"])
	}
	if payload["collection"] != "users" {
		t.Errorf("collection = %v, want users", payload["collection"])
	}
	if payload["passed"] != float64(9) {
		t.Errorf("passed = %v, want 9", payload["passed"])
	}
}

func TestAllEventTypes_Defined(t *testing.T) {
	types := []string{
		EventDataChanged,
		EventRequestCompleted,
		EventTestCompleted,
		EventApprovalPending,
		EventStressTick,
		EventStressCompleted,
	}

	for _, typ := range types {
		if typ == "" {
			t.Errorf("event type constant is empty")
		}
	}

	if len(types) != 6 {
		t.Errorf("expected 6 event types, got %d", len(types))
	}
}

func TestPayloadTypes_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
	}{
		{"DataChanged", DataChangedPayload{Source: "environments"}},
		{"RequestCompleted", RequestCompletedPayload{ReqID: "r1", Status: 201, Time: 100}},
		{"TestCompleted", TestCompletedPayload{Collection: "c1", Passed: 5, Failed: 0, Total: 5, Duration: 800}},
		{"ApprovalPending", ApprovalPendingPayload{ActionID: "a1", ActionType: "delete", Details: "removing env"}},
		{"StressTick", StressTickPayload{Elapsed: 10, RPS: 500.5, P95: 12.3, ErrorRate: 0.01, ActiveUsers: 50}},
		{"StressCompleted", StressCompletedPayload{Scenario: "load-test", Summary: map[string]int{"total": 5000}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewEvent("test.type", tt.payload)
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty JSON")
			}
		})
	}
}
