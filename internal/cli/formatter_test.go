package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/internal/cli"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestNewFormatter_ValidFormats(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{cli.FormatJSON, false},
		{cli.FormatTable, false},
		{cli.FormatMinimal, false},
		{"xml", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			_, err := cli.NewFormatter(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFormatter(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestJSONFormatter_Success(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatJSON)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Success(map[string]string{"key": "value"})
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"ok": true`) {
		t.Errorf("expected 'ok: true' in JSON output, got: %s", out)
	}
	if !strings.Contains(out, `"key"`) {
		t.Errorf("expected data key in JSON output, got: %s", out)
	}
}

func TestJSONFormatter_Error(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatJSON)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Fail("TEST_ERROR", "something went wrong")
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"ok": false`) {
		t.Errorf("expected 'ok: false' in JSON output, got: %s", out)
	}
	if !strings.Contains(out, "TEST_ERROR") {
		t.Errorf("expected error code in JSON output, got: %s", out)
	}
}

func TestTableFormatter_Success_Map(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatTable)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Success(map[string]interface{}{"name": "alice", "status": "active"})
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
		t.Errorf("expected KEY/VALUE headers in table output, got: %s", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected 'alice' value in table output, got: %s", out)
	}
}

func TestTableFormatter_Success_List(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatTable)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	data := []interface{}{
		map[string]interface{}{"id": "1", "name": "alice"},
		map[string]interface{}{"id": "2", "name": "bob"},
	}
	env := envelope.Success(data)
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Errorf("expected list rows in table output, got: %s", out)
	}
}

func TestTableFormatter_Error(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatTable)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Fail("NOT_FOUND", "resource not found")
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "NOT_FOUND") {
		t.Errorf("expected error code in table error output, got: %s", out)
	}
}

func TestMinimalFormatter_Success(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatMinimal)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Success(map[string]string{"result": "done"})
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "ok:") {
		t.Errorf("expected minimal output to start with 'ok:', got: %s", out)
	}
}

func TestMinimalFormatter_Error(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatMinimal)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Fail("AUTH_FAILED", "invalid token")
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "error:") {
		t.Errorf("expected minimal error output to start with 'error:', got: %s", out)
	}
	if !strings.Contains(out, "AUTH_FAILED") {
		t.Errorf("expected error code in minimal output, got: %s", out)
	}
}

func TestMinimalFormatter_NilData(t *testing.T) {
	f, err := cli.NewFormatter(cli.FormatMinimal)
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}

	env := envelope.Success(nil)
	var buf bytes.Buffer
	if err := f.Format(&buf, env); err != nil {
		t.Fatalf("Format: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "ok" {
		t.Errorf("expected 'ok' for nil data, got: %q", out)
	}
}
