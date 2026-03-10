package ws

import (
	"context"
	"encoding/json"
	"time"

	"nhooyr.io/websocket"
)

const (
	// sendBufferSize is the capacity of each client's outbound message
	// channel. A full buffer causes the hub to drop the client.
	sendBufferSize = 256

	// writeWait is the maximum time allowed to write a message to the
	// WebSocket connection before it is considered dead.
	writeWait = 10 * time.Second

	// pongWait is the maximum time to wait for a pong response from
	// the client before considering the connection dead.
	pongWait = 70 * time.Second

	// pingPeriod is how often the server sends a ping to the client.
	// Must be less than pongWait to detect dead connections.
	pingPeriod = 60 * time.Second
)

// Client wraps a single WebSocket connection and provides a buffered
// send channel for outbound messages. Each Client has a dedicated write
// pump goroutine that serialises writes to the underlying connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// newClient creates a Client associated with the given hub and connection.
func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
	}
}

// writePump runs in its own goroutine, reading messages from the send
// channel and writing them to the WebSocket connection. It also handles
// periodic ping messages for keepalive. The pump exits when the send
// channel is closed or a write error occurs.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "closing")
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Hub closed the channel — connection is being torn down.
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Write(ctx, websocket.MessageText, msg)
			cancel()

			if err != nil {
				// Write failed — unregister and stop.
				c.hub.unregister <- c
				return
			}

		case <-ticker.C:
			// Send a ping to detect dead connections.
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Ping(ctx)
			cancel()

			if err != nil {
				c.hub.unregister <- c
				return
			}
		}
	}
}

// readPump runs in its own goroutine, draining any messages the client
// sends (notification-only design — all client messages are discarded).
// When the read returns an error (client disconnect), the pump
// unregisters the client and exits.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
	}()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), pongWait)
		_, _, err := c.conn.Read(ctx)
		cancel()

		if err != nil {
			// Client disconnected or read error.
			return
		}
		// Discard the message — notification-only design.
	}
}

// Send enqueues serialised JSON for delivery. Returns false if the
// client's send buffer is full (slow consumer).
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// marshalEvent serialises an Event to JSON bytes. Returns nil on
// encoding failure.
func marshalEvent(event *Event) []byte {
	data, err := json.Marshal(event)
	if err != nil {
		return nil
	}
	return data
}
