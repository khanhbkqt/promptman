package request

import (
	"crypto/tls"
	"net/http/httptrace"
	"time"
)

// timingTrace captures timing data for each phase of an HTTP request using
// the net/http/httptrace package. Each hook records the start/end time of
// its respective phase, and result() computes the final durations in ms.
type timingTrace struct {
	start time.Time

	dnsStart     time.Time
	dnsDone      time.Time
	connectStart time.Time
	connectDone  time.Time
	tlsStart     time.Time
	tlsDone      time.Time
	gotFirstByte time.Time
	bodyDone     time.Time
}

// newTimingTrace creates a timingTrace and returns a configured
// *httptrace.ClientTrace that hooks into the appropriate trace events.
func newTimingTrace() (*httptrace.ClientTrace, *timingTrace) {
	t := &timingTrace{
		start: time.Now(),
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			t.dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			t.dnsDone = time.Now()
		},
		ConnectStart: func(_, _ string) {
			t.connectStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			t.connectDone = time.Now()
		},
		TLSHandshakeStart: func() {
			t.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			t.tlsDone = time.Now()
		},
		GotFirstResponseByte: func() {
			t.gotFirstByte = time.Now()
		},
	}

	return trace, t
}

// done marks the end of body transfer. Must be called after reading the
// response body.
func (t *timingTrace) done() {
	t.bodyDone = time.Now()
}

// result computes the RequestTiming breakdown from the captured timestamps.
// All durations are in milliseconds. Phases that were not observed (e.g.,
// DNS on a connection-reuse, or TLS on plain HTTP) report 0.
func (t *timingTrace) result() *RequestTiming {
	ms := func(d time.Duration) int {
		return int(d.Milliseconds())
	}

	rt := &RequestTiming{
		Total: ms(t.bodyDone.Sub(t.start)),
	}

	// DNS phase.
	if !t.dnsStart.IsZero() && !t.dnsDone.IsZero() {
		rt.DNS = ms(t.dnsDone.Sub(t.dnsStart))
	}

	// TCP connect phase.
	if !t.connectStart.IsZero() && !t.connectDone.IsZero() {
		rt.Connect = ms(t.connectDone.Sub(t.connectStart))
	}

	// TLS handshake phase.
	if !t.tlsStart.IsZero() && !t.tlsDone.IsZero() {
		rt.TLS = ms(t.tlsDone.Sub(t.tlsStart))
	}

	// Time to first byte (from start of request).
	if !t.gotFirstByte.IsZero() {
		rt.TTFB = ms(t.gotFirstByte.Sub(t.start))
	}

	// Body transfer time (first byte → body done).
	if !t.gotFirstByte.IsZero() && !t.bodyDone.IsZero() {
		rt.Transfer = ms(t.bodyDone.Sub(t.gotFirstByte))
	}

	return rt
}
