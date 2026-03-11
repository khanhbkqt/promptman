package stress

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStressOptsJSON(t *testing.T) {
	opts := StressOpts{
		Collection: "users",
		RequestID:  "list",
		Users:      100,
		RampUp:     "10s",
		Duration:   "60s",
		Thresholds: []string{"p95<500ms", "error_rate<5%"},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got StressOpts
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Collection != opts.Collection {
		t.Errorf("Collection = %q, want %q", got.Collection, opts.Collection)
	}
	if got.Users != opts.Users {
		t.Errorf("Users = %d, want %d", got.Users, opts.Users)
	}
	if got.Duration != opts.Duration {
		t.Errorf("Duration = %q, want %q", got.Duration, opts.Duration)
	}
	if len(got.Thresholds) != 2 {
		t.Errorf("Thresholds len = %d, want 2", len(got.Thresholds))
	}
}

func TestStressConfigYAML(t *testing.T) {
	yamlData := `
name: Users API Load Test
scenarios:
  - name: Browse users
    request: users/list
    weight: 70
    thinkTime: 500ms
  - name: Create user
    request: users/create-user
    weight: 30
config:
  users: 200
  rampUp: 30s
  duration: 120s
thresholds:
  p95_latency: 500ms
  error_rate: "5%"
  rps: 100
`

	var cfg StressConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cfg.Name != "Users API Load Test" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Users API Load Test")
	}
	if len(cfg.Scenarios) != 2 {
		t.Fatalf("Scenarios len = %d, want 2", len(cfg.Scenarios))
	}
	if cfg.Scenarios[0].Name != "Browse users" {
		t.Errorf("Scenario[0].Name = %q, want %q", cfg.Scenarios[0].Name, "Browse users")
	}
	if cfg.Scenarios[0].Weight != 70 {
		t.Errorf("Scenario[0].Weight = %d, want 70", cfg.Scenarios[0].Weight)
	}
	if cfg.Scenarios[0].ThinkTime != "500ms" {
		t.Errorf("Scenario[0].ThinkTime = %q, want %q", cfg.Scenarios[0].ThinkTime, "500ms")
	}
	if cfg.Scenarios[1].Weight != 30 {
		t.Errorf("Scenario[1].Weight = %d, want 30", cfg.Scenarios[1].Weight)
	}
	if cfg.Config.Users != 200 {
		t.Errorf("Config.Users = %d, want 200", cfg.Config.Users)
	}
	if cfg.Config.Duration != "120s" {
		t.Errorf("Config.Duration = %q, want %q", cfg.Config.Duration, "120s")
	}
	if cfg.Thresholds.P95Latency != "500ms" {
		t.Errorf("Thresholds.P95Latency = %q, want %q", cfg.Thresholds.P95Latency, "500ms")
	}
	if cfg.Thresholds.RPS != 100 {
		t.Errorf("Thresholds.RPS = %f, want 100", cfg.Thresholds.RPS)
	}
}

func TestStressConfigYAMLRoundTrip(t *testing.T) {
	original := StressConfig{
		Name: "API Load Test",
		Scenarios: []ScenarioItem{
			{Name: "Get users", Request: "users/list", Weight: 60, ThinkTime: "200ms"},
			{Name: "Create user", Request: "users/create", Weight: 40},
		},
		Config: StressParams{Users: 50, RampUp: "5s", Duration: "30s"},
		Thresholds: ThresholdConfig{
			P95Latency: "300ms",
			ErrorRate:  "2%",
			RPS:        50,
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got StressConfig
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name = %q, want %q", got.Name, original.Name)
	}
	if len(got.Scenarios) != len(original.Scenarios) {
		t.Fatalf("Scenarios len = %d, want %d", len(got.Scenarios), len(original.Scenarios))
	}
	if got.Config.Users != original.Config.Users {
		t.Errorf("Config.Users = %d, want %d", got.Config.Users, original.Config.Users)
	}
}

func TestStressReportJSON(t *testing.T) {
	report := StressReport{
		Scenario: "API Test",
		Duration: 60000,
		Summary: StressSummary{
			TotalRequests:   5000,
			RPS:             83.3,
			Latency:         LatencyMetrics{P50: 45, P95: 120, P99: 250},
			ErrorRate:       1.5,
			Throughput:      1024000,
			PeakConnections: 100,
		},
		Thresholds: []ThresholdResult{
			{Name: "p95_latency", Operator: "<", Expected: 500, Actual: 120, Passed: true},
			{Name: "error_rate", Operator: "<", Expected: 5, Actual: 1.5, Passed: true},
		},
		Timeline: []TimelinePoint{
			{Elapsed: 1, RPS: 50, P95: 100, ErrorRate: 0, ActiveUsers: 50},
			{Elapsed: 2, RPS: 80, P95: 115, ErrorRate: 1, ActiveUsers: 100},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got StressReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Scenario != report.Scenario {
		t.Errorf("Scenario = %q, want %q", got.Scenario, report.Scenario)
	}
	if got.Duration != report.Duration {
		t.Errorf("Duration = %d, want %d", got.Duration, report.Duration)
	}
	if got.Summary.TotalRequests != 5000 {
		t.Errorf("TotalRequests = %d, want 5000", got.Summary.TotalRequests)
	}
	if got.Summary.Latency.P50 != 45 {
		t.Errorf("P50 = %d, want 45", got.Summary.Latency.P50)
	}
	if got.Summary.Latency.P95 != 120 {
		t.Errorf("P95 = %d, want 120", got.Summary.Latency.P95)
	}
	if got.Summary.Latency.P99 != 250 {
		t.Errorf("P99 = %d, want 250", got.Summary.Latency.P99)
	}
	if len(got.Thresholds) != 2 {
		t.Fatalf("Thresholds len = %d, want 2", len(got.Thresholds))
	}
	if !got.Thresholds[0].Passed {
		t.Error("Threshold[0].Passed = false, want true")
	}
	if len(got.Timeline) != 2 {
		t.Fatalf("Timeline len = %d, want 2", len(got.Timeline))
	}
	if got.Timeline[1].RPS != 80 {
		t.Errorf("Timeline[1].RPS = %f, want 80", got.Timeline[1].RPS)
	}
}

func TestTimelinePointJSON(t *testing.T) {
	point := TimelinePoint{
		Elapsed:     5.0,
		RPS:         150.5,
		P95:         100,
		ErrorRate:   2.3,
		ActiveUsers: 50,
	}

	data, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got TimelinePoint
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Elapsed != point.Elapsed {
		t.Errorf("Elapsed = %f, want %f", got.Elapsed, point.Elapsed)
	}
	if got.RPS != point.RPS {
		t.Errorf("RPS = %f, want %f", got.RPS, point.RPS)
	}
	if got.ActiveUsers != point.ActiveUsers {
		t.Errorf("ActiveUsers = %d, want %d", got.ActiveUsers, point.ActiveUsers)
	}
}
