package sandbox

import (
	"strings"
	stdtesting "testing"
)

func TestAtob_Btoa_Roundtrip(t *stdtesting.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"hello world", "hello world"},
		{"empty string", ""},
		{"special chars", "café & naïve"},
		{"binary-like", "\x00\x01\x02"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			script := `btoa("` + escapeJS(tt.input) + `")`
			encoded, err := s.Execute(script)
			if err != nil {
				t.Fatalf("btoa error: %v", err)
			}

			// Decode back with atob.
			script2 := `atob("` + encoded.String() + `")`
			decoded, err := s.Execute(script2)
			if err != nil {
				t.Fatalf("atob error: %v", err)
			}
			if decoded.String() != tt.input {
				t.Errorf("roundtrip got %q, want %q", decoded.String(), tt.input)
			}
		})
	}
}

func TestAtob_Invalid(t *stdtesting.T) {
	s := New()
	_, err := s.Execute(`atob("not-valid-base64!!!")`)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestCrypto_MD5(t *stdtesting.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "hello", "5d41402abc4b2a76b9719d911017c592"},
		{"promptman", "promptman", "c221292a8bd7d9ffbe4604a9216d3d77"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			v, err := s.Execute(`crypto.md5("` + tt.input + `")`)
			if err != nil {
				t.Fatalf("crypto.md5 error: %v", err)
			}
			if v.String() != tt.expected {
				t.Errorf("got %q, want %q", v.String(), tt.expected)
			}
		})
	}
}

func TestCrypto_SHA256(t *stdtesting.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello", "hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *stdtesting.T) {
			s := New()
			v, err := s.Execute(`crypto.sha256("` + tt.input + `")`)
			if err != nil {
				t.Fatalf("crypto.sha256 error: %v", err)
			}
			if v.String() != tt.expected {
				t.Errorf("got %q, want %q", v.String(), tt.expected)
			}
		})
	}
}

func TestConsole_Log(t *stdtesting.T) {
	s := New()
	_, err := s.Execute(`console.log("hello", "world")`)
	if err != nil {
		t.Fatalf("console.log error: %v", err)
	}
	c := s.Console()
	if len(c) != 1 {
		t.Fatalf("expected 1 console entry, got %d", len(c))
	}
	if c[0] != "hello world" {
		t.Errorf("got %q, want %q", c[0], "hello world")
	}
}

func TestConsole_Warn(t *stdtesting.T) {
	s := New()
	_, err := s.Execute(`console.warn("caution")`)
	if err != nil {
		t.Fatalf("console.warn error: %v", err)
	}
	c := s.Console()
	if len(c) != 1 || c[0] != "[warn] caution" {
		t.Errorf("got %q, want %q", c, []string{"[warn] caution"})
	}
}

func TestConsole_Error(t *stdtesting.T) {
	s := New()
	_, err := s.Execute(`console.error("fail")`)
	if err != nil {
		t.Fatalf("console.error error: %v", err)
	}
	c := s.Console()
	if len(c) != 1 || c[0] != "[error] fail" {
		t.Errorf("got %q, want %q", c, []string{"[error] fail"})
	}
}

func TestConsole_Multiple(t *stdtesting.T) {
	s := New()
	_, err := s.Execute(`
		console.log("one");
		console.warn("two");
		console.error("three");
		console.log("four");
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := s.Console()
	if len(c) != 4 {
		t.Fatalf("expected 4 entries, got %d: %v", len(c), c)
	}
	expected := []string{"one", "[warn] two", "[error] three", "four"}
	for i, want := range expected {
		if c[i] != want {
			t.Errorf("entry %d: got %q, want %q", i, c[i], want)
		}
	}
}

func TestConsole_NotOnStdout(t *stdtesting.T) {
	// Console output should only be captured, never printed.
	s := New()
	_, _ = s.Execute(`console.log("silent")`)
	if len(s.Console()) != 1 || s.Console()[0] != "silent" {
		t.Errorf("console output not captured correctly: %v", s.Console())
	}
}

// escapeJS performs basic escaping for embedding in JS string literals.
func escapeJS(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\x00", `\x00`,
		"\x01", `\x01`,
		"\x02", `\x02`,
	)
	return r.Replace(s)
}
