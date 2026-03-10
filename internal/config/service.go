package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

// ConfigService manages project configuration from .promptman/config.yaml.
type ConfigService struct {
	mu         sync.RWMutex
	configPath string
	projectDir string
}

// NewConfigService creates a ConfigService rooted at the given project directory.
// The config file is expected at {projectRoot}/.promptman/config.yaml.
func NewConfigService(projectRoot string) *ConfigService {
	return &ConfigService{
		configPath: filepath.Join(projectRoot, ".promptman", "config.yaml"),
		projectDir: filepath.Join(projectRoot, ".promptman"),
	}
}

// Load reads the project configuration from disk.
// If the config file does not exist, it returns DefaultConfig() with no error.
// If the file exists but cannot be parsed, it returns ErrInvalidConfig.
func (s *ConfigService) Load() (*ProjectConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cfg ProjectConfig
	if err := fsutil.ReadYAML(s.configPath, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return &cfg, nil
}

// Save writes the project configuration to disk atomically.
func (s *ConfigService) Save(cfg *ProjectConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := fsutil.WriteYAML(s.configPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// Init creates the .promptman/ directory structure and writes a default config.yaml.
// It is idempotent: calling Init on an already-initialized project is a no-op.
func (s *ConfigService) Init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subdirs := []string{
		"collections",
		"environments",
		"tests",
		"history",
	}
	for _, sub := range subdirs {
		dir := filepath.Join(s.projectDir, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", sub, err)
		}
	}

	cfg := DefaultConfig()
	if err := fsutil.WriteYAML(s.configPath, cfg); err != nil {
		return fmt.Errorf("write default config: %w", err)
	}
	return nil
}
