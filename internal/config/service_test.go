package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigService(t *testing.T) {
	svc := NewConfigService("/tmp/myproject")
	if svc.configPath != "/tmp/myproject/.promptman/config.yaml" {
		t.Errorf("configPath = %q, want %q", svc.configPath, "/tmp/myproject/.promptman/config.yaml")
	}
	if svc.projectDir != "/tmp/myproject/.promptman" {
		t.Errorf("projectDir = %q, want %q", svc.projectDir, "/tmp/myproject/.promptman")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Should return defaults
	def := DefaultConfig()
	if cfg.Daemon.AutoStart != def.Daemon.AutoStart {
		t.Errorf("missing file should return defaults, got AutoStart=%v", cfg.Daemon.AutoStart)
	}
	if cfg.Daemon.ShutdownAfter != def.Daemon.ShutdownAfter {
		t.Errorf("ShutdownAfter = %q, want %q", cfg.Daemon.ShutdownAfter, def.Daemon.ShutdownAfter)
	}
}

func TestLoadExistingFile(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	// Init to create the file
	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Daemon.AutoStart {
		t.Error("Daemon.AutoStart should be true")
	}
	if cfg.History.RetentionDays != 30 {
		t.Errorf("History.RetentionDays = %d, want 30", cfg.History.RetentionDays)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	// Create a malformed config file
	cfgDir := filepath.Join(dir, ".promptman")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Load()
	if err == nil {
		t.Fatal("Load() should return error for malformed YAML")
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	original := DefaultConfig()
	original.Daemon.ShutdownAfter = "1h"
	original.History.RetentionDays = 90
	original.Approval.DefaultMode = "auto"

	if err := svc.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Daemon.ShutdownAfter != "1h" {
		t.Errorf("ShutdownAfter = %q, want %q", loaded.Daemon.ShutdownAfter, "1h")
	}
	if loaded.History.RetentionDays != 90 {
		t.Errorf("RetentionDays = %d, want 90", loaded.History.RetentionDays)
	}
	if loaded.Approval.DefaultMode != "auto" {
		t.Errorf("DefaultMode = %q, want %q", loaded.Approval.DefaultMode, "auto")
	}
}

func TestInitCreatesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	expectedDirs := []string{
		".promptman",
		".promptman/collections",
		".promptman/environments",
		".promptman/tests",
		".promptman/history",
	}
	for _, d := range expectedDirs {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %q not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", d)
		}
	}

	// Config file should exist
	cfgPath := filepath.Join(dir, ".promptman", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("config.yaml not created: %v", err)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	if err := svc.Init(); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	if err := svc.Init(); err != nil {
		t.Fatalf("second Init() error = %v (should be idempotent)", err)
	}

	// Verify config is still valid after double-init
	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() after double Init() error = %v", err)
	}
	if !cfg.Daemon.AutoStart {
		t.Error("config should still have defaults after double Init()")
	}
}
