package daemon

import (
	"sync"
	"testing"

	tmod "github.com/khanhnguyen/promptman/internal/testing"
)

func TestResultStore_StoreAndGet(t *testing.T) {
	store := NewResultStore(5)

	r1 := &tmod.TestResult{RunID: "run-1", Collection: "users"}
	r2 := &tmod.TestResult{RunID: "run-2", Collection: "health"}

	store.Store(r1)
	store.Store(r2)

	got, ok := store.Get("run-1")
	if !ok || got.RunID != "run-1" {
		t.Errorf("Get(run-1) = %v, %v; want run-1, true", got, ok)
	}

	got, ok = store.Get("run-2")
	if !ok || got.RunID != "run-2" {
		t.Errorf("Get(run-2) = %v, %v; want run-2, true", got, ok)
	}

	_, ok = store.Get("unknown")
	if ok {
		t.Error("Get(unknown) should return false")
	}
}

func TestResultStore_Latest_NewestFirst(t *testing.T) {
	store := NewResultStore(5)

	for i := 1; i <= 3; i++ {
		store.Store(&tmod.TestResult{RunID: "run-" + string(rune('0'+i))})
	}

	latest := store.Latest(3)
	if len(latest) != 3 {
		t.Fatalf("Latest(3) len = %d; want 3", len(latest))
	}
	// Newest first.
	if latest[0].RunID != "run-3" {
		t.Errorf("latest[0].RunID = %q; want run-3", latest[0].RunID)
	}
	if latest[2].RunID != "run-1" {
		t.Errorf("latest[2].RunID = %q; want run-1", latest[2].RunID)
	}
}

func TestResultStore_CircularOverflow(t *testing.T) {
	store := NewResultStore(3)

	// Store 5 items in a buffer of size 3.
	for i := 1; i <= 5; i++ {
		store.Store(&tmod.TestResult{RunID: "run-" + string(rune('0'+i))})
	}

	if store.Len() != 3 {
		t.Errorf("Len() = %d; want 3", store.Len())
	}

	// Should retain runs 3, 4, 5 (oldest two evicted).
	_, ok := store.Get("run-1")
	if ok {
		t.Error("run-1 should have been evicted")
	}
	_, ok = store.Get("run-2")
	if ok {
		t.Error("run-2 should have been evicted")
	}

	got, ok := store.Get("run-5")
	if !ok || got.RunID != "run-5" {
		t.Error("run-5 should still be present")
	}
}

func TestResultStore_LatestExceedsCount(t *testing.T) {
	store := NewResultStore(10)

	store.Store(&tmod.TestResult{RunID: "run-1"})

	latest := store.Latest(5)
	if len(latest) != 1 {
		t.Errorf("Latest(5) with 1 stored = %d; want 1", len(latest))
	}
}

func TestResultStore_DefaultCapacity(t *testing.T) {
	store := NewResultStore(0)
	if store.capacity != defaultResultCapacity {
		t.Errorf("capacity = %d; want %d", store.capacity, defaultResultCapacity)
	}
}

func TestResultStore_ConcurrentAccess(t *testing.T) {
	store := NewResultStore(10)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.Store(&tmod.TestResult{RunID: "run-" + string(rune(id))})
			store.Latest(5)
			store.Get("run-1")
			store.Len()
		}(i)
	}
	wg.Wait()

	if store.Len() > 10 {
		t.Errorf("Len() = %d; should not exceed capacity 10", store.Len())
	}
}
