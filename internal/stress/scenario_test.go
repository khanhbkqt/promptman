package stress

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseOpts_ValidInput(t *testing.T) {
	tests := []struct {
		name       string
		opts       *StressOpts
		wantUsers  int
		wantRampUp string
	}{
		{
			"basic opts",
			&StressOpts{Collection: "users", RequestID: "list", Users: 100, Duration: "60s", RampUp: "10s"},
			100, "10s",
		},
		{
			"no ramp-up defaults to 0s",
			&StressOpts{Collection: "users", RequestID: "list", Users: 50, Duration: "30s"},
			50, "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarios, params, err := ParseOpts(tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(scenarios) != 1 {
				t.Fatalf("expected 1 scenario, got %d", len(scenarios))
			}

			s := scenarios[0]
			if s.CollectionID != tt.opts.Collection {
				t.Errorf("collectionID = %q, want %q", s.CollectionID, tt.opts.Collection)
			}
			if s.RequestID != tt.opts.RequestID {
				t.Errorf("requestID = %q, want %q", s.RequestID, tt.opts.RequestID)
			}
			if s.Weight != 100 {
				t.Errorf("weight = %d, want 100", s.Weight)
			}
			if params.Users != tt.wantUsers {
				t.Errorf("users = %d, want %d", params.Users, tt.wantUsers)
			}
			if params.RampUp != tt.wantRampUp {
				t.Errorf("rampUp = %q, want %q", params.RampUp, tt.wantRampUp)
			}
			if params.Duration != tt.opts.Duration {
				t.Errorf("duration = %q, want %q", params.Duration, tt.opts.Duration)
			}
		})
	}
}

func TestParseOpts_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		opts *StressOpts
	}{
		{"missing collection", &StressOpts{RequestID: "list", Users: 10, Duration: "10s"}},
		{"missing requestID", &StressOpts{Collection: "users", Users: 10, Duration: "10s"}},
		{"zero users", &StressOpts{Collection: "users", RequestID: "list", Users: 0, Duration: "10s"}},
		{"negative users", &StressOpts{Collection: "users", RequestID: "list", Users: -1, Duration: "10s"}},
		{"missing duration", &StressOpts{Collection: "users", RequestID: "list", Users: 10}},
		{"invalid duration", &StressOpts{Collection: "users", RequestID: "list", Users: 10, Duration: "abc"}},
		{"invalid rampUp", &StressOpts{Collection: "users", RequestID: "list", Users: 10, Duration: "10s", RampUp: "xyz"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseOpts(tt.opts)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsDomainError(err, "INVALID_CONFIG") {
				t.Errorf("expected INVALID_CONFIG error, got %v", err)
			}
		})
	}
}

func TestParseConfig_ValidYAML(t *testing.T) {
	yaml := `
name: API Load Test
scenarios:
  - name: Browse users
    request: users/list
    weight: 70
    thinkTime: 500ms
  - name: Create user
    request: users/create
    weight: 30

config:
  users: 200
  rampUp: 30s
  duration: 120s
`
	path := writeTestYAML(t, yaml)

	cfg, scenarios, err := ParseConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "API Load Test" {
		t.Errorf("name = %q, want %q", cfg.Name, "API Load Test")
	}
	if cfg.Config.Users != 200 {
		t.Errorf("users = %d, want 200", cfg.Config.Users)
	}

	if len(scenarios) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(scenarios))
	}

	// Check first scenario.
	s0 := scenarios[0]
	if s0.Name != "Browse users" {
		t.Errorf("scenario[0].name = %q, want %q", s0.Name, "Browse users")
	}
	if s0.CollectionID != "users" {
		t.Errorf("scenario[0].collectionID = %q, want %q", s0.CollectionID, "users")
	}
	if s0.RequestID != "list" {
		t.Errorf("scenario[0].requestID = %q, want %q", s0.RequestID, "list")
	}
	if s0.Weight != 70 {
		t.Errorf("scenario[0].weight = %d, want 70", s0.Weight)
	}
	if s0.ThinkTime != 500*time.Millisecond {
		t.Errorf("scenario[0].thinkTime = %v, want 500ms", s0.ThinkTime)
	}

	// Check second scenario.
	s1 := scenarios[1]
	if s1.Weight != 30 {
		t.Errorf("scenario[1].weight = %d, want 30", s1.Weight)
	}
	if s1.ThinkTime != 0 {
		t.Errorf("scenario[1].thinkTime = %v, want 0", s1.ThinkTime)
	}
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			"zero users",
			`
name: test
scenarios:
  - name: s1
    request: c/r
    weight: 100
config:
  users: 0
  duration: 10s
`,
			"INVALID_CONFIG",
		},
		{
			"missing duration",
			`
name: test
scenarios:
  - name: s1
    request: c/r
    weight: 100
config:
  users: 10
`,
			"INVALID_CONFIG",
		},
		{
			"no scenarios",
			`
name: test
config:
  users: 10
  duration: 10s
`,
			"INVALID_CONFIG",
		},
		{
			"weights not 100",
			`
name: test
scenarios:
  - name: s1
    request: c/r1
    weight: 50
  - name: s2
    request: c/r2
    weight: 30
config:
  users: 10
  duration: 10s
`,
			"INVALID_SCENARIO",
		},
		{
			"invalid request ref no slash",
			`
name: test
scenarios:
  - name: s1
    request: noslash
    weight: 100
config:
  users: 10
  duration: 10s
`,
			"INVALID_SCENARIO",
		},
		{
			"invalid thinkTime",
			`
name: test
scenarios:
  - name: s1
    request: c/r
    weight: 100
    thinkTime: notaduration
config:
  users: 10
  duration: 10s
`,
			"INVALID_SCENARIO",
		},
		{
			"zero weight",
			`
name: test
scenarios:
  - name: s1
    request: c/r
    weight: 0
config:
  users: 10
  duration: 10s
`,
			"INVALID_SCENARIO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestYAML(t, tt.yaml)
			_, _, err := ParseConfig(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsDomainError(err, tt.wantErr) {
				t.Errorf("expected %s error, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestParseConfig_NonexistentFile(t *testing.T) {
	_, _, err := ParseConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseRequestRef(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		wantCol string
		wantReq string
		wantErr bool
	}{
		{"simple", "users/list", "users", "list", false},
		{"nested path", "users/admin/list", "users", "admin/list", false},
		{"empty ref", "", "", "", true},
		{"no slash", "noslash", "", "", true},
		{"empty collection", "/request", "", "", true},
		{"empty request", "collection/", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, req, err := parseRequestRef(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if err == nil {
				if col != tt.wantCol {
					t.Errorf("collectionID = %q, want %q", col, tt.wantCol)
				}
				if req != tt.wantReq {
					t.Errorf("requestID = %q, want %q", req, tt.wantReq)
				}
			}
		})
	}
}

// writeTestYAML writes a YAML string to a temp file and returns the path.
func writeTestYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.stress.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing yaml: %v", err)
	}
	return path
}
