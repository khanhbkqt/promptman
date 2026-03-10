package config

import (
	"sync"
	"testing"
)

func BenchmarkLoad(b *testing.B) {
	dir := b.TempDir()
	svc := NewConfigService(dir)

	// Init to create a real config file
	if err := svc.Init(); err != nil {
		b.Fatalf("Init() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Load()
		if err != nil {
			b.Fatalf("Load() error = %v", err)
		}
	}
}

func BenchmarkLoadMissingFile(b *testing.B) {
	dir := b.TempDir()
	svc := NewConfigService(dir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Load()
		if err != nil {
			b.Fatalf("Load() error = %v", err)
		}
	}
}

func TestConcurrentLoadSave(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigService(dir)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 50

	// Spawn readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cfg, err := svc.Load()
				if err != nil {
					t.Errorf("Load() error = %v", err)
					return
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
					return
				}
			}
		}()
	}

	// Spawn writers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cfg := DefaultConfig()
				cfg.History.RetentionDays = j
				if err := svc.Save(cfg); err != nil {
					t.Errorf("Save() error = %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()
}
