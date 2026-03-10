package ws

import (
	"net/http"

	"nhooyr.io/websocket"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// UpgradeHandler returns an http.HandlerFunc that upgrades an HTTP
// connection to WebSocket. The token query parameter is validated
// before the upgrade — invalid or missing tokens receive a 401
// response without upgrading.
//
// After a successful upgrade the handler creates a Client, registers
// it with the Hub, and starts the read and write pump goroutines.
func UpgradeHandler(hub *Hub, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate token from query parameter before upgrading.
		provided := r.URL.Query().Get("token")
		if provided == "" {
			envelope.WriteError(w, http.StatusUnauthorized,
				envelope.CodeUnauthorized, "missing token query parameter")
			return
		}
		if provided != token {
			envelope.WriteError(w, http.StatusUnauthorized,
				envelope.CodeUnauthorized, "invalid token")
			return
		}

		// Upgrade HTTP → WebSocket.
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// No origin check needed — localhost only.
		})
		if err != nil {
			// websocket.Accept already wrote the HTTP error response.
			return
		}

		client := newClient(hub, conn)
		hub.Register(client)

		// Start read and write pumps in their own goroutines.
		go client.writePump()
		go client.readPump()
	}
}

// Registrar implements daemon.RouteRegistrar and mounts the WebSocket
// upgrade handler onto the daemon's HTTP mux.
type Registrar struct {
	hub   *Hub
	token string
}

// NewRegistrar creates a Registrar that will mount the WS endpoint
// using the given hub and authentication token.
func NewRegistrar(hub *Hub, token string) *Registrar {
	return &Registrar{hub: hub, token: token}
}

// RegisterRoutes mounts the WebSocket upgrade handler at
// GET <prefix>ws (e.g. /api/v1/ws).
func (reg *Registrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"ws", UpgradeHandler(reg.hub, reg.token))
}
