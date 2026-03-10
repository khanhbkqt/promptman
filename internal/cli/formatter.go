package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// Formatter renders an envelope.Envelope to an io.Writer in a specific format.
// Implementations are returned by NewFormatter based on the --format flag value.
type Formatter interface {
	// Format writes the envelope to w in the formatter's output style.
	Format(w io.Writer, env *envelope.Envelope) error
}

// NewFormatter returns a Formatter matching the given format string.
// Accepts FormatJSON, FormatTable, or FormatMinimal.
// Returns an error for unknown format values.
func NewFormatter(format string) (Formatter, error) {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}, nil
	case FormatTable:
		return &TableFormatter{}, nil
	case FormatMinimal:
		return &MinimalFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format %q: must be json, table, or minimal", format)
	}
}

// JSONFormatter outputs the full envelope.Envelope as indented JSON.
type JSONFormatter struct{}

// Format writes the envelope as pretty-printed JSON.
func (f *JSONFormatter) Format(w io.Writer, env *envelope.Envelope) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}
	return nil
}

// TableFormatter renders the envelope data as an aligned ASCII table using
// text/tabwriter from the standard library (no external dependencies).
// If the data is a map, it renders key-value rows.
// If the data is a slice of maps, it renders rows with auto-detected headers.
// Falls back to JSON for complex nested structures.
type TableFormatter struct{}

// Format writes the envelope as a table, or falls back to compact JSON.
func (f *TableFormatter) Format(w io.Writer, env *envelope.Envelope) error {
	if !env.OK {
		// For errors, always show code + message.
		fmt.Fprintf(w, "ERROR\t%s: %s\n", env.Error.Code, env.Error.Message)
		return nil
	}

	if env.Data == nil {
		fmt.Fprintln(w, "OK\t(no data)")
		return nil
	}

	// Convert data to a generic map/slice via JSON round-trip.
	raw, err := json.Marshal(env.Data)
	if err != nil {
		return fmt.Errorf("serialising data for table: %w", err)
	}

	// Try to parse as []interface{} (list of records).
	var list []interface{}
	if err := json.Unmarshal(raw, &list); err == nil {
		return renderList(w, list)
	}

	// Try to parse as map[string]interface{} (single record).
	var record map[string]interface{}
	if err := json.Unmarshal(raw, &record); err == nil {
		return renderRecord(w, record)
	}

	// Scalar or complex structure — print as raw JSON.
	fmt.Fprintf(w, "%s\n", raw)
	return nil
}

// renderList renders a slice of items as a table with auto-detected columns.
func renderList(w io.Writer, list []interface{}) error {
	if len(list) == 0 {
		fmt.Fprintln(w, "(empty list)")
		return nil
	}

	// Collect all column names from the first record.
	headers := collectHeaders(list[0])

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Print header row.
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	fmt.Fprintln(tw, strings.Join(makeUnderlines(headers), "\t"))

	// Print each row.
	for _, item := range list {
		row := extractValues(item, headers)
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return nil
}

// renderRecord renders a single map as two-column key/value rows.
func renderRecord(w io.Writer, record map[string]interface{}) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	fmt.Fprintln(tw, "KEY\tVALUE")
	fmt.Fprintln(tw, "---\t-----")
	for k, v := range record {
		fmt.Fprintf(tw, "%s\t%v\n", k, formatValue(v))
	}
	return nil
}

// collectHeaders extracts field names from the first item in a list.
func collectHeaders(item interface{}) []string {
	m, ok := item.(map[string]interface{})
	if !ok {
		return []string{"value"}
	}
	headers := make([]string, 0, len(m))
	for k := range m {
		headers = append(headers, k)
	}
	return headers
}

// extractValues extracts values from an item in column-header order.
func extractValues(item interface{}, headers []string) []string {
	m, ok := item.(map[string]interface{})
	if !ok {
		return []string{fmt.Sprintf("%v", item)}
	}
	vals := make([]string, len(headers))
	for i, h := range headers {
		vals[i] = formatValue(m[h])
	}
	return vals
}

// makeUnderlines returns dashes under each header for visual separation.
func makeUnderlines(headers []string) []string {
	lines := make([]string, len(headers))
	for i, h := range headers {
		lines[i] = strings.Repeat("-", len(h))
	}
	return lines
}

// formatValue formats a single value for table display.
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice:
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// MinimalFormatter renders only the essential outcome: ok status and a summary.
// For errors it shows the error code and message.
// For success it shows a compact representation of the data.
type MinimalFormatter struct{}

// Format writes a single-line minimal representation of the envelope.
func (f *MinimalFormatter) Format(w io.Writer, env *envelope.Envelope) error {
	if !env.OK {
		fmt.Fprintf(w, "error: %s — %s\n", env.Error.Code, env.Error.Message)
		return nil
	}

	if env.Data == nil {
		fmt.Fprintln(w, "ok")
		return nil
	}

	// Compact JSON of just the data field.
	raw, err := json.Marshal(env.Data)
	if err != nil {
		return fmt.Errorf("marshalling data for minimal output: %w", err)
	}
	fmt.Fprintf(w, "ok: %s\n", raw)
	return nil
}
