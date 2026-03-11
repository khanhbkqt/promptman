package stress

import "time"

// StressOpts configures a stress test from CLI flags (quick mode).
type StressOpts struct {
	Collection string   `json:"collection"`           // collection path
	RequestID  string   `json:"requestId"`            // request ID within the collection
	Users      int      `json:"users"`                // concurrent virtual users
	RampUp     string   `json:"rampUp,omitempty"`     // ramp-up duration, e.g. "10s"
	Duration   string   `json:"duration"`             // total test duration, e.g. "60s"
	Thresholds []string `json:"thresholds,omitempty"` // e.g. "p95<500ms", "error_rate<5%"
}

// StressConfig represents a YAML config file for complex stress scenarios.
type StressConfig struct {
	Name       string          `yaml:"name"       json:"name"`                           // test name
	Scenarios  []ScenarioItem  `yaml:"scenarios"  json:"scenarios"`                      // request scenarios
	Config     StressParams    `yaml:"config"     json:"config"`                         // execution params
	Thresholds ThresholdConfig `yaml:"thresholds,omitempty" json:"thresholds,omitempty"` // pass/fail thresholds
}

// StressParams holds the execution parameters nested within StressConfig.
type StressParams struct {
	Users    int    `yaml:"users"    json:"users"`            // concurrent virtual users
	RampUp   string `yaml:"rampUp"   json:"rampUp,omitempty"` // ramp-up duration
	Duration string `yaml:"duration" json:"duration"`         // total test duration
}

// ScenarioItem defines a single request scenario within a stress config.
type ScenarioItem struct {
	Name      string `yaml:"name"      json:"name"`                          // human-readable name
	Request   string `yaml:"request"   json:"request"`                       // collectionId/requestId
	Weight    int    `yaml:"weight"    json:"weight"`                        // traffic weight percentage
	ThinkTime string `yaml:"thinkTime,omitempty" json:"thinkTime,omitempty"` // pause between requests
}

// StressReport holds the complete results of a stress test run.
type StressReport struct {
	Scenario   string            `json:"scenario"`             // scenario name
	Duration   int64             `json:"duration"`             // total duration in ms
	Summary    StressSummary     `json:"summary"`              // aggregate metrics
	Thresholds []ThresholdResult `json:"thresholds,omitempty"` // threshold eval results
	Timeline   []TimelinePoint   `json:"timeline"`             // per-second snapshots
}

// StressSummary holds aggregate metrics for a completed stress test.
type StressSummary struct {
	TotalRequests   int            `json:"totalRequests"`   // total HTTP requests executed
	RPS             float64        `json:"rps"`             // requests per second
	Latency         LatencyMetrics `json:"latency"`         // latency percentiles
	ErrorRate       float64        `json:"errorRate"`       // error percentage (0–100)
	Throughput      int64          `json:"throughput"`      // bytes per second
	PeakConnections int            `json:"peakConnections"` // max concurrent connections
}

// LatencyMetrics holds latency percentile values in milliseconds.
type LatencyMetrics struct {
	P50 int64 `json:"p50"` // 50th percentile (median) in ms
	P95 int64 `json:"p95"` // 95th percentile in ms
	P99 int64 `json:"p99"` // 99th percentile in ms
}

// TimelinePoint captures a per-second metrics snapshot for live charting.
type TimelinePoint struct {
	Elapsed     float64 `json:"elapsed"`     // seconds since test start
	RPS         float64 `json:"rps"`         // requests per second in this window
	P95         int64   `json:"p95"`         // 95th percentile latency in ms
	ErrorRate   float64 `json:"errorRate"`   // error percentage in this window
	ActiveUsers int     `json:"activeUsers"` // active virtual users
}

// ThresholdResult records the outcome of a single threshold evaluation.
type ThresholdResult struct {
	Name     string  `json:"name"`     // e.g. "p95_latency"
	Operator string  `json:"operator"` // e.g. "<"
	Expected float64 `json:"expected"` // threshold limit value
	Actual   float64 `json:"actual"`   // measured value
	Passed   bool    `json:"passed"`   // whether the threshold was met
}

// ThresholdConfig defines configurable pass/fail thresholds from YAML.
type ThresholdConfig struct {
	P95Latency string  `yaml:"p95_latency,omitempty" json:"p95Latency,omitempty"` // e.g. "500ms"
	ErrorRate  string  `yaml:"error_rate,omitempty"  json:"errorRate,omitempty"`  // e.g. "5%"
	RPS        float64 `yaml:"rps,omitempty"         json:"rps,omitempty"`        // minimum RPS
}

// RequestResult captures the outcome of a single HTTP request execution.
// Used internally by MetricsCollector.Record().
type RequestResult struct {
	Latency    time.Duration // request round-trip time
	StatusCode int           // HTTP status code
	BytesSent  int64         // response body bytes
	Error      bool          // whether the request was an error (5xx or connection failure)
}
