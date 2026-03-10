package config

// ProjectConfig holds the entire project-level configuration
// read from .promptman/config.yaml.
type ProjectConfig struct {
	Daemon   DaemonConfig   `yaml:"daemon"`
	History  HistoryConfig  `yaml:"history,omitempty"`
	Testing  TestingConfig  `yaml:"testing,omitempty"`
	Approval ApprovalConfig `yaml:"approval,omitempty"`
}

// DaemonConfig controls the background daemon lifecycle.
type DaemonConfig struct {
	AutoStart     bool   `yaml:"autoStart"`
	ShutdownAfter string `yaml:"shutdownAfter"`
}

// HistoryConfig controls request history logging.
type HistoryConfig struct {
	Enabled       bool `yaml:"enabled"`
	RetentionDays int  `yaml:"retentionDays"`
}

// TestingConfig controls the functional test runner defaults.
type TestingConfig struct {
	Timeout     string `yaml:"timeout"`
	TestTimeout string `yaml:"testTimeout"`
}

// ApprovalConfig controls the CLI approval gate behaviour.
type ApprovalConfig struct {
	DefaultMode string            `yaml:"defaultMode"`
	ByAction    map[string]string `yaml:"byAction,omitempty"`
	ByEnv       map[string]string `yaml:"byEnv,omitempty"`
}
