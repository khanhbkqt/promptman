package request

import (
	"fmt"
	"io"
	"net/http"
)

// captureResponse reads the HTTP response and builds a Response struct
// with status, headers, body, and timing data. The caller must have called
// timingTrace.done() before invoking this function.
func captureResponse(resp *http.Response, timing *timingTrace, reqID, method, resolvedURL string) (*Response, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	headers := extractHeaders(resp.Header)

	return &Response{
		RequestID: reqID,
		Method:    method,
		URL:       resolvedURL,
		Status:    resp.StatusCode,
		Headers:   headers,
		Body:      string(body),
		Timing:    timing.result(),
	}, nil
}

// extractHeaders flattens http.Header (multi-value) into a single-value map.
// When a header has multiple values, they are joined with ", ".
func extractHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for key, vals := range h {
		if len(vals) == 1 {
			result[key] = vals[0]
		} else {
			result[key] = joinValues(vals)
		}
	}
	return result
}

// joinValues joins multiple header values with ", ".
func joinValues(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	s := vals[0]
	for _, v := range vals[1:] {
		s += ", " + v
	}
	return s
}
