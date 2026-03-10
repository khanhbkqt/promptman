package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// ResponseDisplay holds the decoded fields from a single HTTP response.
// This is used by the response-specific formatters to render output
// without coupling to the internal/request package.
type ResponseDisplay struct {
	RequestID string            `json:"requestId"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Error     string            `json:"error,omitempty"`
	Timing    *TimingDisplay    `json:"timing,omitempty"`
}

// TimingDisplay holds timing breakdown in milliseconds.
type TimingDisplay struct {
	DNS      int `json:"dns"`
	Connect  int `json:"connect"`
	TLS      int `json:"tls"`
	TTFB     int `json:"ttfb"`
	Transfer int `json:"transfer"`
	Total    int `json:"total"`
}

// FormatResponseTable writes an HTTP response in table format with
// status line, timing breakdown, headers, and body.
func FormatResponseTable(w io.Writer, resp *ResponseDisplay) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Status line.
	statusEmoji := statusEmoji(resp.Status)
	fmt.Fprintf(tw, "%s %d\t%s %s\n", statusEmoji, resp.Status, resp.Method, resp.URL)

	// Timing breakdown (if present).
	if resp.Timing != nil {
		fmt.Fprintln(tw)
		fmt.Fprintln(tw, "PHASE\tDURATION")
		fmt.Fprintln(tw, "-----\t--------")
		fmt.Fprintf(tw, "DNS\t%dms\n", resp.Timing.DNS)
		fmt.Fprintf(tw, "Connect\t%dms\n", resp.Timing.Connect)
		if resp.Timing.TLS > 0 {
			fmt.Fprintf(tw, "TLS\t%dms\n", resp.Timing.TLS)
		}
		fmt.Fprintf(tw, "TTFB\t%dms\n", resp.Timing.TTFB)
		fmt.Fprintf(tw, "Transfer\t%dms\n", resp.Timing.Transfer)
		fmt.Fprintf(tw, "Total\t%dms\n", resp.Timing.Total)
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flushing timing table: %w", err)
	}

	// Headers.
	if len(resp.Headers) > 0 {
		fmt.Fprintln(w)
		htw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(htw, "HEADER\tVALUE")
		fmt.Fprintln(htw, "------\t-----")
		for k, v := range resp.Headers {
			fmt.Fprintf(htw, "%s\t%s\n", k, v)
		}
		if err := htw.Flush(); err != nil {
			return fmt.Errorf("flushing headers table: %w", err)
		}
	}

	// Body.
	if resp.Body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, resp.Body)
	}

	// Error (if any).
	if resp.Error != "" {
		fmt.Fprintf(w, "\n⚠ Error: %s\n", resp.Error)
	}

	return nil
}

// FormatResponseMinimal writes a compact single-line summary followed by the body.
// Format: STATUS METHOD URL (Xms)
func FormatResponseMinimal(w io.Writer, resp *ResponseDisplay) error {
	timing := ""
	if resp.Timing != nil {
		timing = fmt.Sprintf(" (%dms)", resp.Timing.Total)
	}
	fmt.Fprintf(w, "%d %s %s%s\n", resp.Status, resp.Method, resp.URL, timing)

	if resp.Body != "" {
		fmt.Fprintln(w, resp.Body)
	}

	if resp.Error != "" {
		fmt.Fprintf(w, "error: %s\n", resp.Error)
	}

	return nil
}

// FormatRunResponse renders an envelope containing an HTTP response in the specified format.
// For table and minimal formats, it decodes the response data and uses the response-specific
// formatters. For JSON, it delegates to the standard Formatter.
func FormatRunResponse(w io.Writer, format string, env *envelope.Envelope) error {
	// JSON format: delegate to standard formatter.
	if format == FormatJSON {
		f, err := NewFormatter(FormatJSON)
		if err != nil {
			return err
		}
		return f.Format(w, env)
	}

	// For error envelopes, use the standard formatter.
	if !env.OK {
		f, err := NewFormatter(format)
		if err != nil {
			return err
		}
		return f.Format(w, env)
	}

	// Decode response data.
	raw, err := json.Marshal(env.Data)
	if err != nil {
		return fmt.Errorf("serialising response data: %w", err)
	}

	// Try single response.
	var single ResponseDisplay
	if err := json.Unmarshal(raw, &single); err == nil && single.Method != "" {
		if format == FormatTable {
			return FormatResponseTable(w, &single)
		}
		return FormatResponseMinimal(w, &single)
	}

	// Try collection response (array).
	var list []ResponseDisplay
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return formatCollectionResponses(w, format, list)
	}

	// Fallback to standard formatter.
	f, err := NewFormatter(format)
	if err != nil {
		return err
	}
	return f.Format(w, env)
}

// formatCollectionResponses renders multiple HTTP responses with separators.
func formatCollectionResponses(w io.Writer, format string, responses []ResponseDisplay) error {
	for i, resp := range responses {
		if i > 0 {
			fmt.Fprintln(w, strings.Repeat("─", 60))
		}

		r := resp // avoid loop variable capture
		if format == FormatTable {
			if err := FormatResponseTable(w, &r); err != nil {
				return err
			}
		} else {
			if err := FormatResponseMinimal(w, &r); err != nil {
				return err
			}
		}
	}
	return nil
}

// statusEmoji returns a colored status indicator for the given HTTP status code.
func statusEmoji(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "✓"
	case status >= 300 && status < 400:
		return "→"
	case status >= 400 && status < 500:
		return "✗"
	default:
		return "⚠"
	}
}
