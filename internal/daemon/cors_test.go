package daemon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		expectCORS     bool
		expectStatus   int
		expectNext     bool
	}{
		{
			name:         "localhost:3000 origin",
			method:       "GET",
			origin:       "http://localhost:3000",
			expectCORS:   true,
			expectStatus: http.StatusOK,
			expectNext:   true,
		},
		{
			name:         "127.0.0.1 origin",
			method:       "GET",
			origin:       "http://127.0.0.1",
			expectCORS:   true,
			expectStatus: http.StatusOK,
			expectNext:   true,
		},
		{
			name:         "external origin",
			method:       "GET",
			origin:       "https://google.com",
			expectCORS:   false,
			expectStatus: http.StatusOK,
			expectNext:   true,
		},
		{
			name:         "preflight OPTIONS",
			method:       "OPTIONS",
			origin:       "http://localhost:5173",
			expectCORS:   true,
			expectStatus: http.StatusNoContent,
			expectNext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := CORSMiddleware()(next)
			req := httptest.NewRequest(tt.method, "http://example.com/api/v1/status", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled to be %v, got %v", tt.expectNext, nextCalled)
			}

			originHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if originHeader != tt.origin {
					t.Errorf("expected CORS origin %s, got %s", tt.origin, originHeader)
				}
				if w.Header().Get("Access-Control-Allow-Methods") == "" {
					t.Error("expected Access-Control-Allow-Methods header")
				}
			} else {
				if originHeader != "" {
					t.Errorf("expected no CORS origin, got %s", originHeader)
				}
			}
		})
	}
}

func TestIsLocalhostOrigin(t *testing.T) {
	tests := []struct {
		origin string
		want   bool
	}{
		{"http://localhost:3000", true},
		{"http://127.0.0.1:5173", true},
		{"http://localhost", true},
		{"http://127.0.0.1", true},
		{"https://localhost:3000", false}, // we only specified http:// in prefix matching
		{"http://localghost:3000", false},
		{"http://google.com", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isLocalhostOrigin(tt.origin); got != tt.want {
			t.Errorf("isLocalhostOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
		}
	}
}
