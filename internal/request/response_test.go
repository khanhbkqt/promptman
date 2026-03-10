package request

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCaptureResponse_SuccessfulResponse(t *testing.T) {
	body := `{"status":"ok"}`
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": {"application/json"},
			"X-Request-Id": {"abc-123"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}

	start := time.Now()
	tt := &timingTrace{
		start:        start,
		gotFirstByte: start.Add(10 * time.Millisecond),
		bodyDone:     start.Add(15 * time.Millisecond),
	}

	result, err := captureResponse(resp, tt, "get-users", "GET", "https://example.com/users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != 200 {
		t.Errorf("Status = %d, want 200", result.Status)
	}
	if result.Body != body {
		t.Errorf("Body = %q, want %q", result.Body, body)
	}
	if result.RequestID != "get-users" {
		t.Errorf("RequestID = %q, want %q", result.RequestID, "get-users")
	}
	if result.Method != "GET" {
		t.Errorf("Method = %q, want %q", result.Method, "GET")
	}
	if result.URL != "https://example.com/users" {
		t.Errorf("URL = %q, want %q", result.URL, "https://example.com/users")
	}
	if result.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want %q", result.Headers["Content-Type"], "application/json")
	}
	if result.Headers["X-Request-Id"] != "abc-123" {
		t.Errorf("X-Request-Id = %q, want %q", result.Headers["X-Request-Id"], "abc-123")
	}
	if result.Timing == nil {
		t.Fatal("Timing should not be nil")
	}
	if result.Timing.Total != 15 {
		t.Errorf("Timing.Total = %d, want 15", result.Timing.Total)
	}
}

func TestCaptureResponse_EmptyBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 204,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	start := time.Now()
	tt := &timingTrace{
		start:    start,
		bodyDone: start.Add(5 * time.Millisecond),
	}

	result, err := captureResponse(resp, tt, "delete-user", "DELETE", "https://example.com/users/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != 204 {
		t.Errorf("Status = %d, want 204", result.Status)
	}
	if result.Body != "" {
		t.Errorf("Body = %q, want empty", result.Body)
	}
}

func TestCaptureResponse_MultiValueHeaders(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Set-Cookie": {"session=abc", "theme=dark"},
		},
		Body: io.NopCloser(strings.NewReader("")),
	}

	start := time.Now()
	tt := &timingTrace{
		start:    start,
		bodyDone: start.Add(1 * time.Millisecond),
	}

	result, err := captureResponse(resp, tt, "login", "POST", "https://example.com/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "session=abc, theme=dark"
	if result.Headers["Set-Cookie"] != want {
		t.Errorf("Set-Cookie = %q, want %q", result.Headers["Set-Cookie"], want)
	}
}

func TestExtractHeaders_EmptyHeaders(t *testing.T) {
	h := http.Header{}
	result := extractHeaders(h)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestJoinValues_Empty(t *testing.T) {
	result := joinValues(nil)
	if result != "" {
		t.Errorf("joinValues(nil) = %q, want empty", result)
	}
}

func TestJoinValues_Single(t *testing.T) {
	result := joinValues([]string{"one"})
	if result != "one" {
		t.Errorf("joinValues single = %q, want %q", result, "one")
	}
}

func TestJoinValues_Multiple(t *testing.T) {
	result := joinValues([]string{"a", "b", "c"})
	want := "a, b, c"
	if result != want {
		t.Errorf("joinValues = %q, want %q", result, want)
	}
}
