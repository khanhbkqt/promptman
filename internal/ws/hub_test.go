package ws

import (
	"sync"
	"testing"
	"time"
)

func TestHub_RegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// Create a mock client with a send channel we can read from.
	client := &Client{
		hub:  hub,
		send: make(chan []byte, sendBufferSize),
	}

	hub.Register(client)

	// Give the hub goroutine time to process the register.
	time.Sleep(10 * time.Millisecond)

	// Broadcast an event.
	event := NewEvent(EventDataChanged, DataChangedPayload{Source: "collections"})
	hub.Broadcast(event)

	// Read the broadcasted message.
	select {
	case msg := <-client.send:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, sendBufferSize),
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	// After unregister, the send channel should be closed.
	_, ok := <-client.send
	if ok {
		t.Error("expected send channel to be closed after unregister")
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const numClients = 10
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = &Client{
			hub:  hub,
			send: make(chan []byte, sendBufferSize),
		}
		hub.Register(clients[i])
	}

	time.Sleep(20 * time.Millisecond)

	// Broadcast an event.
	event := NewEvent(EventRequestCompleted, RequestCompletedPayload{
		ReqID: "req-1", Status: 200, Time: 50,
	})
	hub.Broadcast(event)

	// All clients should receive the message.
	for i, client := range clients {
		select {
		case msg := <-client.send:
			if len(msg) == 0 {
				t.Errorf("client %d: expected non-empty message", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d: timeout waiting for broadcast", i)
		}
	}
}

func TestHub_SlowConsumerDropped(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	// Create a client with a tiny buffer.
	client := &Client{
		hub:  hub,
		send: make(chan []byte, 1),
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Fill the send buffer.
	client.send <- []byte("filler")

	// Now broadcast — client buffer is full, should be dropped.
	event := NewEvent(EventDataChanged, DataChangedPayload{Source: "config"})
	hub.Broadcast(event)

	time.Sleep(20 * time.Millisecond)

	// The send channel should be closed (client was dropped).
	// Drain the filler message first.
	<-client.send

	_, ok := <-client.send
	if ok {
		t.Error("expected send channel to be closed for slow consumer")
	}
}

func TestHub_GracefulShutdown(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		hub:  hub,
		send: make(chan []byte, sendBufferSize),
	}
	client2 := &Client{
		hub:  hub,
		send: make(chan []byte, sendBufferSize),
	}
	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	// Shutdown should close all client channels and return.
	hub.Shutdown()

	// Both clients' send channels should be closed.
	_, ok1 := <-client1.send
	_, ok2 := <-client2.send
	if ok1 {
		t.Error("client1 send channel should be closed after shutdown")
	}
	if ok2 {
		t.Error("client2 send channel should be closed after shutdown")
	}
}

func TestHub_ShutdownIdempotent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Multiple shutdowns should not panic.
	hub.Shutdown()

	// Second shutdown in a goroutine to detect panics/deadlocks.
	done := make(chan struct{})
	go func() {
		hub.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("second Shutdown() deadlocked")
	}
}

func TestHub_ConcurrentOperations(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	var wg sync.WaitGroup
	const numOps = 50

	// Concurrent registers.
	clients := make([]*Client, numOps)
	for i := 0; i < numOps; i++ {
		clients[i] = &Client{
			hub:  hub,
			send: make(chan []byte, sendBufferSize),
		}
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			hub.Register(c)
		}(clients[i])
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	// Concurrent broadcasts.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Broadcast(NewEvent(EventStressTick, StressTickPayload{
				RPS: 100, P95: 5, ErrorRate: 0, ActiveUsers: 10,
			}))
		}()
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	// Concurrent unregisters.
	for _, c := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			hub.Unregister(c)
		}(c)
	}

	wg.Wait()
}
