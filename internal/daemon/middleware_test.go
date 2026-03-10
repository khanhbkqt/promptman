package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestAuthMiddleware(t *testing.T) {
	const validToken = "test-secret-token-12345"

	// A simple handler that the middleware will wrap.
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := AuthMiddleware(validToken)
	handler := middleware(okHandler)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantCode   string // envelope error code, empty if success
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantCode:   envelope.CodeUnauthorized,
		},
		{
			name:       "no bearer prefix",
			authHeader: validToken,
			wantStatus: http.StatusUnauthorized,
			wantCode:   envelope.CodeUnauthorized,
		},
		{
			name:       "wrong token",
			authHeader: "Bearer wrong-token-value",
			wantStatus: http.StatusUnauthorized,
			wantCode:   envelope.CodeUnauthorized,
		},
		{
			name:       "empty bearer",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
			wantCode:   envelope.CodeUnauthorized,
		},
		{
			name:       "basic auth instead of bearer",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
			wantCode:   envelope.CodeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantCode != "" {
				var env envelope.Envelope
				if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
					t.Fatalf("decoding envelope: %v", err)
				}
				if env.OK {
					t.Error("expected ok=false for unauthorized request")
				}
				if env.Error == nil {
					t.Fatal("expected error in envelope")
				}
				if env.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", env.Error.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestIdleResetMiddleware(t *testing.T) {
	// Create a manager with a long idle timeout so the timer is active.
	mgr := NewManager(WithIdleTimeout(10 * time.Minute))

	dir := t.TempDir()
	_, err := mgr.Start(dir)
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}
	defer func() { _ = mgr.Stop() }()

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := IdleResetMiddleware(mgr)
	handler := middleware(okHandler)

	// Make a request — this should not panic and should pass through.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify the manager is still running (timer was reset, not fired).
	if !mgr.IsRunning() {
		t.Error("manager should still be running after idle reset")
	}
}
