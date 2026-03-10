package fsutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadJSON reads a JSON file and unmarshals it into the given output.
func ReadJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read json %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse json %s: %w", path, err)
	}
	return nil
}

// WriteJSON marshals data as indented JSON and writes it atomically to a file.
// Parent directories are created automatically.
func WriteJSON(path string, data any) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	bytes = append(bytes, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	return AtomicWrite(path, bytes)
}

// AppendJSONL appends a single JSON entry as a newline-delimited JSON line.
// The file is created if it doesn't exist. Parent directories are created automatically.
func AppendJSONL(path string, entry any) error {
	bytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal jsonl entry: %w", err)
	}
	bytes = append(bytes, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open jsonl %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(bytes); err != nil {
		return fmt.Errorf("write jsonl %s: %w", path, err)
	}
	return nil
}
