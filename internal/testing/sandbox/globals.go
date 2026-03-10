package sandbox

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// injectGlobals registers the curated standard library into the VM:
// atob/btoa, crypto.md5/sha256, and console.log/warn/error capture.
func (s *Sandbox) injectGlobals() {
	s.injectBase64()
	s.injectCrypto()
	s.injectConsole()
}

// injectBase64 adds atob (decode) and btoa (encode) global functions.
func (s *Sandbox) injectBase64() {
	_ = s.vm.Set("atob", func(call goja.FunctionCall) goja.Value {
		encoded := call.Argument(0).String()
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			panic(s.vm.NewGoError(fmt.Errorf("atob: invalid base64: %w", err)))
		}
		return s.vm.ToValue(string(decoded))
	})

	_ = s.vm.Set("btoa", func(call goja.FunctionCall) goja.Value {
		raw := call.Argument(0).String()
		return s.vm.ToValue(base64.StdEncoding.EncodeToString([]byte(raw)))
	})
}

// injectCrypto adds a crypto object with md5(str) and sha256(str) methods.
func (s *Sandbox) injectCrypto() {
	obj := s.vm.NewObject()

	_ = obj.Set("md5", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()
		sum := md5.Sum([]byte(data))
		return s.vm.ToValue(fmt.Sprintf("%x", sum))
	})

	_ = obj.Set("sha256", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()
		sum := sha256.Sum256([]byte(data))
		return s.vm.ToValue(fmt.Sprintf("%x", sum))
	})

	_ = s.vm.Set("crypto", obj)
}

// injectConsole adds console.log, console.warn, and console.error functions
// that capture output into s.console instead of printing to stdout.
func (s *Sandbox) injectConsole() {
	obj := s.vm.NewObject()

	makeLogger := func(level string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = arg.String()
			}
			msg := strings.Join(parts, " ")
			if level != "log" {
				msg = fmt.Sprintf("[%s] %s", level, msg)
			}
			s.console = append(s.console, msg)
			return goja.Undefined()
		}
	}

	_ = obj.Set("log", makeLogger("log"))
	_ = obj.Set("warn", makeLogger("warn"))
	_ = obj.Set("error", makeLogger("error"))

	_ = s.vm.Set("console", obj)
}
