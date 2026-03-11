package stress

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// RequestExecutor executes a single HTTP request and returns the raw
// metrics needed for stress test recording. This interface decouples
// the stress module from internal/request.
type RequestExecutor interface {
	Execute(ctx context.Context, collectionID, requestID string) (statusCode int, latencyUs int64, bodySize int64, err error)
}

// scheduler manages goroutine spawning with ramp-up timing. It spawns
// workers at a gradual rate over the ramp-up period so that the system
// is not overwhelmed with sudden load.
type scheduler struct {
	totalUsers  int
	rampUp      time.Duration
	activeUsers int64 // atomic counter
	wg          sync.WaitGroup
}

// newScheduler creates a scheduler for the given user count and ramp-up.
func newScheduler(totalUsers int, rampUp time.Duration) *scheduler {
	return &scheduler{
		totalUsers: totalUsers,
		rampUp:     rampUp,
	}
}

// Start spawns workers gradually over the ramp-up period. Each worker
// receives the provided workerFn which runs its request loop. Start
// blocks until all workers have been spawned (not until they finish).
// Call Wait() to wait for all workers to exit.
func (s *scheduler) Start(ctx context.Context, workerFn func(ctx context.Context)) {
	if s.totalUsers <= 0 {
		return
	}

	// No ramp-up: spawn all at once.
	if s.rampUp <= 0 {
		for i := 0; i < s.totalUsers; i++ {
			s.spawnWorker(ctx, workerFn)
		}
		return
	}

	// Calculate spawn interval: how often to spawn a batch of workers.
	// We spawn users/rampUpSeconds workers per tick, at least 1 per tick.
	rampUpSecs := s.rampUp.Seconds()
	if rampUpSecs <= 0 {
		rampUpSecs = 1
	}

	usersPerSecond := float64(s.totalUsers) / rampUpSecs
	if usersPerSecond < 1 {
		usersPerSecond = 1
	}

	// Tick interval: spawn at 10Hz for smoother ramp-up, or less frequently
	// if we have very few users.
	tickInterval := 100 * time.Millisecond
	usersPerTick := usersPerSecond / 10.0
	if usersPerTick < 1 {
		// Fewer than 10 users/sec: tick once per user.
		tickInterval = time.Duration(float64(time.Second) / usersPerSecond)
		usersPerTick = 1
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	spawned := 0
	accumulator := 0.0

	for spawned < s.totalUsers {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			accumulator += usersPerTick
			batch := int(accumulator)
			if batch <= 0 {
				continue
			}
			accumulator -= float64(batch)

			for i := 0; i < batch && spawned < s.totalUsers; i++ {
				s.spawnWorker(ctx, workerFn)
				spawned++
			}
		}
	}
}

// spawnWorker launches a single worker goroutine.
func (s *scheduler) spawnWorker(ctx context.Context, fn func(ctx context.Context)) {
	s.wg.Add(1)
	atomic.AddInt64(&s.activeUsers, 1)

	go func() {
		defer s.wg.Done()
		defer atomic.AddInt64(&s.activeUsers, -1)
		fn(ctx)
	}()
}

// ActiveUsers returns the current number of active worker goroutines.
func (s *scheduler) ActiveUsers() int {
	return int(atomic.LoadInt64(&s.activeUsers))
}

// Wait blocks until all worker goroutines have exited.
func (s *scheduler) Wait() {
	s.wg.Wait()
}
