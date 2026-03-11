package stress

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduler_SpawnAllAtOnce(t *testing.T) {
	s := newScheduler(10, 0)

	var count int64
	ctx, cancel := context.WithCancel(context.Background())

	s.Start(ctx, func(ctx context.Context) {
		atomic.AddInt64(&count, 1)
		<-ctx.Done()
	})

	// All should be spawned immediately.
	if got := s.ActiveUsers(); got != 10 {
		t.Errorf("active users = %d, want 10", got)
	}

	cancel()
	s.Wait()

	if got := atomic.LoadInt64(&count); got != 10 {
		t.Errorf("total workers run = %d, want 10", got)
	}
}

func TestScheduler_RampUpGradual(t *testing.T) {
	s := newScheduler(20, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.Start(ctx, func(ctx context.Context) {
			<-ctx.Done()
		})
		close(done)
	}()

	// After 200ms, should have roughly 4 users (20 users / 1s * 0.2s).
	// Allow wide tolerance since timing is imprecise.
	time.Sleep(200 * time.Millisecond)
	active := s.ActiveUsers()
	if active >= 20 {
		t.Errorf("after 200ms: active = %d, should not have all 20 yet", active)
	}

	// Wait for ramp-up to complete.
	<-done

	if got := s.ActiveUsers(); got != 20 {
		t.Errorf("after ramp-up: active = %d, want 20", got)
	}

	cancel()
	s.Wait()
}

func TestScheduler_CancellationStopsSpawning(t *testing.T) {
	s := newScheduler(100, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Start(ctx, func(ctx context.Context) {
			<-ctx.Done()
		})
		close(done)
	}()

	// Cancel after 100ms — should have far fewer than 100 workers.
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-done
	s.Wait()

	if got := int(atomic.LoadInt64(&s.activeUsers)); got >= 50 {
		t.Errorf("spawned %d workers in 100ms of 5s ramp-up — too many", got)
	}
}

func TestScheduler_ZeroUsers(t *testing.T) {
	s := newScheduler(0, time.Second)

	ctx := context.Background()
	s.Start(ctx, func(ctx context.Context) {
		t.Error("worker should not be called for 0 users")
	})

	s.Wait()
}

func TestScheduler_SingleUser(t *testing.T) {
	s := newScheduler(1, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Start(ctx, func(ctx context.Context) {
			<-ctx.Done()
		})
		close(done)
	}()

	<-done

	if got := s.ActiveUsers(); got != 1 {
		t.Errorf("active users = %d, want 1", got)
	}

	cancel()
	s.Wait()
}

func TestScheduler_ActiveUsersDecrementsOnExit(t *testing.T) {
	s := newScheduler(5, 0)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx, func(ctx context.Context) {
		<-ctx.Done()
	})

	if got := s.ActiveUsers(); got != 5 {
		t.Errorf("before cancel: active = %d, want 5", got)
	}

	cancel()
	s.Wait()

	if got := s.ActiveUsers(); got != 0 {
		t.Errorf("after wait: active = %d, want 0", got)
	}
}
