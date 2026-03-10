package config

// DefaultConfig returns a ProjectConfig populated with the spec-compliant
// default values from M10 section 5.
func DefaultConfig() *ProjectConfig {
	return &ProjectConfig{
		Daemon: DaemonConfig{
			AutoStart:     true,
			ShutdownAfter: "30m",
		},
		History: HistoryConfig{
			Enabled:       true,
			RetentionDays: 30,
		},
		Testing: TestingConfig{
			Timeout:     "120s",
			TestTimeout: "10s",
		},
		Approval: ApprovalConfig{
			DefaultMode: "prompt",
			ByAction: map[string]string{
				"read":        "auto",
				"write":       "auto",
				"execute":     "prompt",
				"destructive": "prompt",
			},
			ByEnv: map[string]string{
				"dev":     "auto",
				"staging": "prompt",
				"prod":    "prompt",
			},
		},
	}
}
