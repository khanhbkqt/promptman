package daemon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// apiPrefix is the versioned API path prefix for all REST endpoints.
	apiPrefix = "/api/v1/"

	// shutdownDrainTimeout is the maximum time to wait for in-flight
	// requests to complete during graceful shutdown.
	shutdownDrainTimeout = 5 * time.Second
)

// RouteRegistrar is implemented by modules that want to mount their
// handlers onto the daemon's HTTP server. Each module registers its
// routes under the given prefix (e.g. "/api/v1/").
type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux, prefix string)
}

// Server wraps an http.Server and integrates with the daemon Manager.
// It provides a versioned REST API router with middleware support.
type Server struct {
	mu         sync.Mutex
	httpSrv    *http.Server
	mgr        *Manager
	mux        *http.ServeMux
	running    bool
	doneCh     chan struct{} // closed when ListenAndServe returns
	registrars []RouteRegistrar
}

// NewServer creates a new Server associated with the given Manager.
// Route registrars can be added to mount module-specific handlers.
func NewServer(mgr *Manager, registrars ...RouteRegistrar) *Server {
	return &Server{
		mgr:        mgr,
		registrars: registrars,
	}
}

// Start begins serving HTTP on the given address (e.g. "127.0.0.1:48721").
// It sets up the versioned router, middleware chain, and starts listening
// in a background goroutine. The method returns once the server is
// listening or an error occurs during startup.
func (s *Server) Start(addr string, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrDaemonAlreadyRunning.Wrap("server already running")
	}

	s.mux = http.NewServeMux()

	// Register core daemon endpoints.
	s.mux.HandleFunc("GET "+apiPrefix+"status", StatusHandler(s.mgr))
	s.mux.HandleFunc("POST "+apiPrefix+"shutdown", ShutdownHandler(s))

	// Let modules register their routes.
	for _, r := range s.registrars {
		r.RegisterRoutes(s.mux, apiPrefix)
	}

	// Build middleware chain: auth → idle-reset → mux.
	var handler http.Handler = s.mux
	handler = IdleResetMiddleware(s.mgr)(handler)
	handler = AuthMiddleware(token)(handler)

	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Create a listener so we can detect binding errors synchronously.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("binding to %s: %w", addr, err)
	}

	s.doneCh = make(chan struct{})
	s.running = true

	go func() {
		defer close(s.doneCh)
		// Serve blocks until the server is shut down.
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			// Log but don't propagate — the server is shutting down.
			fmt.Printf("[daemon] server error: %v\n", err)
		}
	}()

	return nil
}

// Shutdown gracefully stops the HTTP server, draining in-flight requests
// within the shutdownDrainTimeout (5 seconds).
func (s *Server) Shutdown() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	srv := s.httpSrv
	doneCh := s.doneCh
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownDrainTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("draining in-flight requests: %w", err)
	}

	// Wait for the serve goroutine to finish.
	<-doneCh

	return nil
}

// IsRunning reports whether the HTTP server is currently active.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
