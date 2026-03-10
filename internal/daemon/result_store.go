package daemon

import (
	"sync"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// defaultResultCapacity is the default number of test results to keep
// in the in-memory circular buffer.
const defaultResultCapacity = 10

// ResultStore provides thread-safe in-memory storage for test results
// using a circular buffer. The most recent N results are retained.
type ResultStore struct {
	mu       sync.Mutex
	results  []*testing.TestResult
	capacity int
	index    int // next write position
	count    int // total items stored (up to capacity)
}

// NewResultStore creates a ResultStore with the given capacity.
// If capacity is <= 0, the default capacity (10) is used.
func NewResultStore(capacity int) *ResultStore {
	if capacity <= 0 {
		capacity = defaultResultCapacity
	}
	return &ResultStore{
		results:  make([]*testing.TestResult, capacity),
		capacity: capacity,
	}
}

// Store adds a test result to the buffer, overwriting the oldest
// entry when the buffer is full.
func (s *ResultStore) Store(result *testing.TestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[s.index] = result
	s.index = (s.index + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
}

// Latest returns the most recent n results, ordered newest-first.
// If n > stored count, all stored results are returned.
func (s *ResultStore) Latest(n int) []*testing.TestResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n > s.count {
		n = s.count
	}
	if n <= 0 {
		return nil
	}

	out := make([]*testing.TestResult, n)
	for i := 0; i < n; i++ {
		// Walk backwards from the most recently written position.
		pos := (s.index - 1 - i + s.capacity) % s.capacity
		out[i] = s.results[pos]
	}
	return out
}

// Get retrieves a test result by its RunID. Returns nil and false
// if no result with that ID is found.
func (s *ResultStore) Get(runID string) (*testing.TestResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i < s.count; i++ {
		pos := (s.index - 1 - i + s.capacity) % s.capacity
		if s.results[pos] != nil && s.results[pos].RunID == runID {
			return s.results[pos], true
		}
	}
	return nil, false
}

// Len returns the number of results currently stored.
func (s *ResultStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}
