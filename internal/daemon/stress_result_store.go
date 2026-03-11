package daemon

import (
	"sync"

	"github.com/khanhnguyen/promptman/internal/stress"
)

// defaultStressResultCapacity is the number of stress test results
// to retain in the in-memory circular buffer.
const defaultStressResultCapacity = 10

// StressResultStore provides thread-safe in-memory storage for stress
// test results using a circular buffer. The most recent N results are
// retained. It follows the same pattern as ResultStore for testing.
type StressResultStore struct {
	mu       sync.Mutex
	results  []*stressResultEntry
	capacity int
	index    int // next write position
	count    int // total items stored (up to capacity)
}

// stressResultEntry wraps a stress report with a job ID for retrieval.
type stressResultEntry struct {
	JobID  string              `json:"jobId"`
	Report *stress.StressReport `json:"report"`
}

// NewStressResultStore creates a StressResultStore with the given capacity.
// If capacity is <= 0, the default capacity (10) is used.
func NewStressResultStore(capacity int) *StressResultStore {
	if capacity <= 0 {
		capacity = defaultStressResultCapacity
	}
	return &StressResultStore{
		results:  make([]*stressResultEntry, capacity),
		capacity: capacity,
	}
}

// Store adds a stress test result to the buffer, overwriting the
// oldest entry when full.
func (s *StressResultStore) Store(jobID string, report *stress.StressReport) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[s.index] = &stressResultEntry{JobID: jobID, Report: report}
	s.index = (s.index + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
}

// Get retrieves a stress result by job ID. Returns nil and false if
// no result with that ID is found.
func (s *StressResultStore) Get(jobID string) (*stress.StressReport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i < s.count; i++ {
		pos := (s.index - 1 - i + s.capacity) % s.capacity
		if s.results[pos] != nil && s.results[pos].JobID == jobID {
			return s.results[pos].Report, true
		}
	}
	return nil, false
}

// Latest returns the most recent n results, ordered newest-first.
func (s *StressResultStore) Latest(n int) []*stressResultEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n > s.count {
		n = s.count
	}
	if n <= 0 {
		return nil
	}

	out := make([]*stressResultEntry, n)
	for i := 0; i < n; i++ {
		pos := (s.index - 1 - i + s.capacity) % s.capacity
		out[i] = s.results[pos]
	}
	return out
}
