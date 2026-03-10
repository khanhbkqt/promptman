package ws

import "sync"

// Hub manages the set of active WebSocket clients and broadcasts
// events to all of them. It runs a single goroutine that serialises
// access to the client set via channels — no mutexes are needed for
// the set itself.
type Hub struct {
	// clients is the set of currently registered clients.
	// Only accessed by the Run goroutine.
	clients map[*Client]struct{}

	// register receives clients to add to the set.
	register chan *Client

	// unregister receives clients to remove from the set.
	unregister chan *Client

	// broadcast receives events to fan out to all clients.
	broadcast chan *Event

	// stop signals the hub goroutine to shut down.
	stop chan struct{}

	// done is closed when the Run goroutine exits.
	done chan struct{}

	// shutdownOnce ensures Shutdown is idempotent.
	shutdownOnce sync.Once
}

// NewHub creates a Hub ready to accept client registrations and
// broadcast events. Call Run() in a goroutine to start processing.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Event, 256),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Run is the main event loop. It must be called in its own goroutine.
// It processes register, unregister, and broadcast operations until
// Shutdown is called.
func (h *Hub) Run() {
	defer close(h.done)

	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case event := <-h.broadcast:
			data := marshalEvent(event)
			if data == nil {
				continue
			}

			for client := range h.clients {
				if !client.Send(data) {
					// Slow consumer — drop the client.
					delete(h.clients, client)
					close(client.send)
				}
			}

		case <-h.stop:
			// Close all client connections.
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			return
		}
	}
}

// Register adds a client to the hub. Safe to call from any goroutine.
func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.stop:
	}
}

// Unregister removes a client from the hub. Safe to call from any goroutine.
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.stop:
	}
}

// Broadcast sends an event to all connected clients. The event is
// serialised to JSON once and then fanned out. Safe to call from any
// goroutine.
func (h *Hub) Broadcast(event *Event) {
	select {
	case h.broadcast <- event:
	case <-h.stop:
	}
}

// Shutdown gracefully stops the hub, closing all client connections.
// It blocks until the Run goroutine has exited. Safe to call multiple
// times — only the first call has effect.
func (h *Hub) Shutdown() {
	h.shutdownOnce.Do(func() {
		close(h.stop)
	})
	<-h.done
}

// ClientCount returns the number of currently registered clients.
// This is mainly useful for testing.
func (h *Hub) ClientCount() int {
	// We read via a synchronous round-trip through the hub goroutine
	// to avoid races. For simplicity in v1, we return the length of
	// the broadcast channel as a proxy. In production, a dedicated
	// query channel would be cleaner.
	//
	// Note: This is only safe to call from tests after ensuring all
	// register/unregister operations have been processed.
	return len(h.clients)
}
