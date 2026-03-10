package envelope

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccess(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"nil data", nil},
		{"string data", "hello"},
		{"int data", 42},
		{"map data", map[string]string{"key": "value"}},
		{"slice data", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := Success(tt.data)
			if !env.OK {
				t.Error("expected OK to be true")
			}
			if env.Error != nil {
				t.Error("expected Error to be nil")
			}
		})
	}
}

func TestFail(t *testing.T) {
	env := Fail("TEST_CODE", "test message")

	if env.OK {
		t.Error("expected OK to be false")
	}
	if env.Data != nil {
		t.Error("expected Data to be nil")
	}
	if env.Error == nil {
		t.Fatal("expected Error to not be nil")
	}
	if env.Error.Code != "TEST_CODE" {
		t.Errorf("expected code TEST_CODE, got %s", env.Error.Code)
	}
	if env.Error.Message != "test message" {
		t.Errorf("expected message 'test message', got %s", env.Error.Message)
	}
	if env.Error.Details != nil {
		t.Error("expected Details to be nil")
	}
}

func TestFailWithDetails(t *testing.T) {
	details := map[string]string{"field": "name", "issue": "required"}
	env := FailWithDetails("VALIDATION", "validation failed", details)

	if env.OK {
		t.Error("expected OK to be false")
	}
	if env.Error == nil {
		t.Fatal("expected Error to not be nil")
	}
	if env.Error.Details == nil {
		t.Error("expected Details to not be nil")
	}
}

func TestSuccessJSON(t *testing.T) {
	env := Success(map[string]int{"count": 5})
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if result["ok"] != true {
		t.Error("expected ok=true in JSON")
	}
	if result["error"] != nil {
		t.Error("expected error=null in JSON")
	}
	dataMap, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if dataMap["count"] != float64(5) {
		t.Errorf("expected data.count=5, got %v", dataMap["count"])
	}
}

func TestFailJSON(t *testing.T) {
	env := Fail("NOT_FOUND", "item not found")
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if result["ok"] != false {
		t.Error("expected ok=false in JSON")
	}
	if result["data"] != nil {
		t.Error("expected data=null in JSON")
	}
	errMap, ok := result["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error to be a map")
	}
	if errMap["code"] != "NOT_FOUND" {
		t.Errorf("expected error.code=NOT_FOUND, got %v", errMap["code"])
	}
}

func TestFailWithDetailsJSON_OmitsEmptyDetails(t *testing.T) {
	// FailWithDetails with nil details should omit the details field
	env := FailWithDetails("CODE", "msg", nil)
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	errMap := result["error"].(map[string]any)
	if _, exists := errMap["details"]; exists {
		t.Error("expected details to be omitted when nil")
	}
}

func TestHTTPStatusForCode(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{CodeCollectionNotFound, http.StatusNotFound},
		{CodeRequestNotFound, http.StatusNotFound},
		{CodeEnvNotFound, http.StatusNotFound},
		{CodeNotFound, http.StatusNotFound},
		{CodeEnvNotSet, http.StatusBadRequest},
		{CodeInvalidYAML, http.StatusBadRequest},
		{CodeInvalidRequest, http.StatusBadRequest},
		{CodeInvalidInput, http.StatusBadRequest},
		{CodeImportFailed, http.StatusBadRequest},
		{CodeExportFailed, http.StatusBadRequest},
		{CodeRequestTimeout, http.StatusRequestTimeout},
		{CodeRequestFailed, http.StatusBadGateway},
		{CodeTestExecutionError, http.StatusInternalServerError},
		{CodeInternalError, http.StatusInternalServerError},
		{CodeDaemonBusy, http.StatusTooManyRequests},
		{CodeUnauthorized, http.StatusUnauthorized},
		{"UNKNOWN_CODE", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := HTTPStatusForCode(tt.code)
			if got != tt.expected {
				t.Errorf("HTTPStatusForCode(%s) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestWriteSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	WriteSuccess(w, http.StatusOK, map[string]string{"name": "test"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %s", ct)
	}

	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !env.OK {
		t.Error("expected OK=true")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusNotFound, CodeCollectionNotFound, "collection not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if env.OK {
		t.Error("expected OK=false")
	}
	if env.Error == nil || env.Error.Code != CodeCollectionNotFound {
		t.Error("expected error code COLLECTION_NOT_FOUND")
	}
}

func TestWrapHandler_NoPanic(t *testing.T) {
	handler := WrapHandler(func(w http.ResponseWriter, r *http.Request) {
		WriteSuccess(w, http.StatusOK, "hello")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWrapHandler_PanicRecovery(t *testing.T) {
	handler := WrapHandler(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if env.OK {
		t.Error("expected OK=false after panic")
	}
	if env.Error == nil || env.Error.Code != CodeInternalError {
		t.Error("expected INTERNAL_ERROR code after panic")
	}
}
