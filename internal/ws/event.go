package ws

import "time"

// Event type constants identify the category of a broadcasted event.
// Clients use the Type field to determine how to handle the payload.
const (
	// EventDataChanged is broadcast when a data file on disk changes
	// (collection, environment, test, or config YAML).
	EventDataChanged = "data.changed"

	// EventRequestCompleted is broadcast when an HTTP request execution
	// finishes, carrying the request ID, status code, and timing.
	EventRequestCompleted = "request.completed"

	// EventTestCompleted is broadcast when a test suite run finishes,
	// carrying aggregate pass/fail/total counts and duration.
	EventTestCompleted = "test.completed"

	// EventApprovalPending is broadcast when an AI-triggered action
	// requires human approval before proceeding.
	EventApprovalPending = "approval.pending"

	// EventStressTick is broadcast every second during a stress test,
	// carrying real-time metrics (RPS, p95 latency, error rate).
	EventStressTick = "stress.tick"

	// EventStressCompleted is broadcast when a stress test finishes,
	// carrying the final summary with scenario details.
	EventStressCompleted = "stress.completed"
)

// Event is the message envelope sent to all connected WebSocket clients.
// It carries the event type, a typed payload, and a UTC timestamp.
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	TS      time.Time   `json:"ts"`
}

// NewEvent creates an Event with the given type and payload, setting
// the timestamp to the current UTC time.
func NewEvent(eventType string, payload interface{}) *Event {
	return &Event{
		Type:    eventType,
		Payload: payload,
		TS:      time.Now().UTC(),
	}
}

// DataChangedPayload carries the source of a data change event.
// Source is one of: "collections", "environments", "tests", "config".
type DataChangedPayload struct {
	Source string `json:"source"`
}

// RequestCompletedPayload carries the result of an HTTP request execution.
type RequestCompletedPayload struct {
	ReqID  string `json:"reqId"`
	Status int    `json:"status"`
	Time   int64  `json:"time"` // duration in milliseconds
}

// TestCompletedPayload carries aggregate results of a test suite run.
type TestCompletedPayload struct {
	Collection string `json:"collection"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	Total      int    `json:"total"`
	Duration   int64  `json:"duration"` // milliseconds
}

// ApprovalPendingPayload carries details about an AI action awaiting
// human approval.
type ApprovalPendingPayload struct {
	ActionID   string `json:"actionId"`
	ActionType string `json:"actionType"`
	Details    string `json:"details"`
}

// StressTickPayload carries real-time metrics emitted every second
// during a stress test.
type StressTickPayload struct {
	Elapsed     int64   `json:"elapsed"`     // seconds since start
	RPS         float64 `json:"rps"`         // requests per second
	P95         float64 `json:"p95"`         // 95th percentile latency (ms)
	ErrorRate   float64 `json:"errorRate"`   // 0.0–1.0
	ActiveUsers int     `json:"activeUsers"` // concurrent virtual users
}

// StressCompletedPayload carries the final summary of a completed
// stress test.
type StressCompletedPayload struct {
	Scenario string      `json:"scenario"`
	Summary  interface{} `json:"summary"` // flexible summary object
}
