package request

import (
	"testing"
	"time"
)

func TestTimingTrace_Result(t *testing.T) {
	tt := &timingTrace{
		start: time.Now(),
	}

	// Simulate phases with known durations.
	tt.dnsStart = tt.start.Add(1 * time.Millisecond)
	tt.dnsDone = tt.start.Add(3 * time.Millisecond)
	tt.connectStart = tt.start.Add(3 * time.Millisecond)
	tt.connectDone = tt.start.Add(8 * time.Millisecond)
	tt.tlsStart = tt.start.Add(8 * time.Millisecond)
	tt.tlsDone = tt.start.Add(15 * time.Millisecond)
	tt.gotFirstByte = tt.start.Add(20 * time.Millisecond)
	tt.bodyDone = tt.start.Add(25 * time.Millisecond)

	rt := tt.result()

	if rt.DNS != 2 {
		t.Errorf("DNS = %d, want 2", rt.DNS)
	}
	if rt.Connect != 5 {
		t.Errorf("Connect = %d, want 5", rt.Connect)
	}
	if rt.TLS != 7 {
		t.Errorf("TLS = %d, want 7", rt.TLS)
	}
	if rt.TTFB != 20 {
		t.Errorf("TTFB = %d, want 20", rt.TTFB)
	}
	if rt.Transfer != 5 {
		t.Errorf("Transfer = %d, want 5", rt.Transfer)
	}
	if rt.Total != 25 {
		t.Errorf("Total = %d, want 25", rt.Total)
	}
}

func TestTimingTrace_NoPhasesObserved(t *testing.T) {
	start := time.Now()
	tt := &timingTrace{
		start:    start,
		bodyDone: start.Add(10 * time.Millisecond),
	}

	rt := tt.result()

	// Phases not observed should be 0.
	if rt.DNS != 0 {
		t.Errorf("DNS = %d, want 0", rt.DNS)
	}
	if rt.Connect != 0 {
		t.Errorf("Connect = %d, want 0", rt.Connect)
	}
	if rt.TLS != 0 {
		t.Errorf("TLS = %d, want 0", rt.TLS)
	}
	if rt.TTFB != 0 {
		t.Errorf("TTFB = %d, want 0", rt.TTFB)
	}
	if rt.Transfer != 0 {
		t.Errorf("Transfer = %d, want 0", rt.Transfer)
	}
	if rt.Total != 10 {
		t.Errorf("Total = %d, want 10", rt.Total)
	}
}

func TestTimingTrace_NoTLS(t *testing.T) {
	start := time.Now()
	tt := &timingTrace{
		start:        start,
		dnsStart:     start.Add(1 * time.Millisecond),
		dnsDone:      start.Add(2 * time.Millisecond),
		connectStart: start.Add(2 * time.Millisecond),
		connectDone:  start.Add(4 * time.Millisecond),
		gotFirstByte: start.Add(10 * time.Millisecond),
		bodyDone:     start.Add(12 * time.Millisecond),
	}

	rt := tt.result()

	if rt.TLS != 0 {
		t.Errorf("TLS = %d, want 0 (plain HTTP)", rt.TLS)
	}
	if rt.DNS != 1 {
		t.Errorf("DNS = %d, want 1", rt.DNS)
	}
	if rt.Connect != 2 {
		t.Errorf("Connect = %d, want 2", rt.Connect)
	}
}

func TestTimingTrace_Done(t *testing.T) {
	tt := &timingTrace{start: time.Now()}

	if !tt.bodyDone.IsZero() {
		t.Fatal("bodyDone should be zero before done()")
	}

	tt.done()

	if tt.bodyDone.IsZero() {
		t.Fatal("bodyDone should be set after done()")
	}
}

func TestNewTimingTrace_ReturnsNonNil(t *testing.T) {
	trace, tt := newTimingTrace()

	if trace == nil {
		t.Fatal("expected non-nil ClientTrace")
	}
	if tt == nil {
		t.Fatal("expected non-nil timingTrace")
	}
	if tt.start.IsZero() {
		t.Fatal("start should be set on creation")
	}
}
