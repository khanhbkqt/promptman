package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProjectConfigYAMLRoundTrip(t *testing.T) {
	original := DefaultConfig()

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal back
	var restored ProjectConfig
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify all fields survived round-trip
	assertDaemonConfig(t, original.Daemon, restored.Daemon)
	assertHistoryConfig(t, original.History, restored.History)
	assertTestingConfig(t, original.Testing, restored.Testing)
	assertApprovalConfig(t, original.Approval, restored.Approval)
}

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig()

	// Daemon defaults
	if !cfg.Daemon.AutoStart {
		t.Error("Daemon.AutoStart should default to true")
	}
	if cfg.Daemon.ShutdownAfter != "30m" {
		t.Errorf("Daemon.ShutdownAfter = %q, want %q", cfg.Daemon.ShutdownAfter, "30m")
	}

	// History defaults
	if !cfg.History.Enabled {
		t.Error("History.Enabled should default to true")
	}
	if cfg.History.RetentionDays != 30 {
		t.Errorf("History.RetentionDays = %d, want %d", cfg.History.RetentionDays, 30)
	}

	// Testing defaults
	if cfg.Testing.Timeout != "120s" {
		t.Errorf("Testing.Timeout = %q, want %q", cfg.Testing.Timeout, "120s")
	}
	if cfg.Testing.TestTimeout != "10s" {
		t.Errorf("Testing.TestTimeout = %q, want %q", cfg.Testing.TestTimeout, "10s")
	}

	// Approval defaults
	if cfg.Approval.DefaultMode != "prompt" {
		t.Errorf("Approval.DefaultMode = %q, want %q", cfg.Approval.DefaultMode, "prompt")
	}

	// ByAction map
	expectedActions := map[string]string{
		"read":        "auto",
		"write":       "auto",
		"execute":     "prompt",
		"destructive": "prompt",
	}
	for action, want := range expectedActions {
		got, ok := cfg.Approval.ByAction[action]
		if !ok {
			t.Errorf("ByAction[%q] missing", action)
		} else if got != want {
			t.Errorf("ByAction[%q] = %q, want %q", action, got, want)
		}
	}
	if len(cfg.Approval.ByAction) != len(expectedActions) {
		t.Errorf("ByAction has %d entries, want %d", len(cfg.Approval.ByAction), len(expectedActions))
	}

	// ByEnv map
	expectedEnvs := map[string]string{
		"dev":     "auto",
		"staging": "prompt",
		"prod":    "prompt",
	}
	for env, want := range expectedEnvs {
		got, ok := cfg.Approval.ByEnv[env]
		if !ok {
			t.Errorf("ByEnv[%q] missing", env)
		} else if got != want {
			t.Errorf("ByEnv[%q] = %q, want %q", env, got, want)
		}
	}
	if len(cfg.Approval.ByEnv) != len(expectedEnvs) {
		t.Errorf("ByEnv has %d entries, want %d", len(cfg.Approval.ByEnv), len(expectedEnvs))
	}
}

func TestDefaultConfigReturnsNewInstance(t *testing.T) {
	a := DefaultConfig()
	b := DefaultConfig()
	a.Daemon.AutoStart = false
	if !b.Daemon.AutoStart {
		t.Error("DefaultConfig should return a new instance each call")
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrConfigNotFound == nil {
		t.Error("ErrConfigNotFound should not be nil")
	}
	if ErrInvalidConfig == nil {
		t.Error("ErrInvalidConfig should not be nil")
	}
	if ErrConfigNotFound.Error() == "" {
		t.Error("ErrConfigNotFound should have a message")
	}
	if ErrInvalidConfig.Error() == "" {
		t.Error("ErrInvalidConfig should have a message")
	}
}

// --- assertion helpers ---

func assertDaemonConfig(t *testing.T, want, got DaemonConfig) {
	t.Helper()
	if got.AutoStart != want.AutoStart {
		t.Errorf("Daemon.AutoStart = %v, want %v", got.AutoStart, want.AutoStart)
	}
	if got.ShutdownAfter != want.ShutdownAfter {
		t.Errorf("Daemon.ShutdownAfter = %q, want %q", got.ShutdownAfter, want.ShutdownAfter)
	}
}

func assertHistoryConfig(t *testing.T, want, got HistoryConfig) {
	t.Helper()
	if got.Enabled != want.Enabled {
		t.Errorf("History.Enabled = %v, want %v", got.Enabled, want.Enabled)
	}
	if got.RetentionDays != want.RetentionDays {
		t.Errorf("History.RetentionDays = %d, want %d", got.RetentionDays, want.RetentionDays)
	}
}

func assertTestingConfig(t *testing.T, want, got TestingConfig) {
	t.Helper()
	if got.Timeout != want.Timeout {
		t.Errorf("Testing.Timeout = %q, want %q", got.Timeout, want.Timeout)
	}
	if got.TestTimeout != want.TestTimeout {
		t.Errorf("Testing.TestTimeout = %q, want %q", got.TestTimeout, want.TestTimeout)
	}
}

func assertApprovalConfig(t *testing.T, want, got ApprovalConfig) {
	t.Helper()
	if got.DefaultMode != want.DefaultMode {
		t.Errorf("Approval.DefaultMode = %q, want %q", got.DefaultMode, want.DefaultMode)
	}
	for k, v := range want.ByAction {
		if got.ByAction[k] != v {
			t.Errorf("Approval.ByAction[%q] = %q, want %q", k, got.ByAction[k], v)
		}
	}
	for k, v := range want.ByEnv {
		if got.ByEnv[k] != v {
			t.Errorf("Approval.ByEnv[%q] = %q, want %q", k, got.ByEnv[k], v)
		}
	}
}
