package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestUpgradeHandler_ValidToken(t *testing.T) {
	const token = "test-ws-token"

	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	handler := UpgradeHandler(hub, token)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Convert HTTP URL to WS URL.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dialing WS: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	// Give the hub time to register the client.
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event and verify the client receives it.
	event := NewEvent(EventDataChanged, DataChangedPayload{Source: "collections"})
	hub.Broadcast(event)

	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()

	_, msg, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("reading WS message: %v", err)
	}

	var received Event
	if err := json.Unmarshal(msg, &received); err != nil {
		t.Fatalf("unmarshalling event: %v", err)
	}

	if received.Type != EventDataChanged {
		t.Errorf("event type = %q, want %q", received.Type, EventDataChanged)
	}
}

func TestUpgradeHandler_InvalidToken(t *testing.T) {
	const token = "valid-token"

	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	handler := UpgradeHandler(hub, token)
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=wrong-token"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for invalid token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestUpgradeHandler_MissingToken(t *testing.T) {
	const token = "valid-token"

	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	handler := UpgradeHandler(hub, token)
	server := httptest.NewServer(handler)
	defer server.Close()

	// No token query parameter.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, wsURL, nil)
	if err == nil {
		t.Fatal("expected dial error for missing token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestUpgradeHandler_BroadcastDelivery(t *testing.T) {
	const token = "broadcast-test-token"

	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	handler := UpgradeHandler(hub, token)
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token

	// Connect 3 clients.
	conns := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		cancel()
		if err != nil {
			t.Fatalf("client %d dial: %v", i, err)
		}
		conns[i] = conn
		defer conn.Close(websocket.StatusNormalClosure, "done")
	}

	time.Sleep(50 * time.Millisecond)

	// Broadcast.
	hub.Broadcast(NewEvent(EventTestCompleted, TestCompletedPayload{
		Collection: "apis",
		Passed:     5,
		Failed:     0,
		Total:      5,
		Duration:   1200,
	}))

	// All 3 clients should receive the event.
	for i, conn := range conns {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, msg, err := conn.Read(ctx)
		cancel()

		if err != nil {
			t.Fatalf("client %d read: %v", i, err)
		}

		var event Event
		if err := json.Unmarshal(msg, &event); err != nil {
			t.Fatalf("client %d unmarshal: %v", i, err)
		}

		if event.Type != EventTestCompleted {
			t.Errorf("client %d: type = %q, want %q", i, event.Type, EventTestCompleted)
		}
	}
}
