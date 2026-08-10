package cli

import (
	"io"
	"os"
	"testing"
)

func TestDuplicateTTYCreatesIndependentFile(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp(t.TempDir(), "duplicate-tty-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString("duplicate me"); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("rewind temp file: %v", err)
	}

	dup, err := duplicateTTY(file)
	if err != nil {
		t.Fatalf("duplicateTTY: %v", err)
	}
	defer dup.Close()

	if got, want := dup.Name(), file.Name(); got != want {
		t.Fatalf("duplicateTTY name = %q, want %q", got, want)
	}
	got, err := io.ReadAll(dup)
	if err != nil {
		t.Fatalf("read duplicate: %v", err)
	}
	if string(got) != "duplicate me" {
		t.Fatalf("read duplicate = %q, want %q", got, "duplicate me")
	}

	if err := file.Close(); err != nil {
		t.Fatalf("close original: %v", err)
	}
	if _, err := dup.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek duplicate after original close: %v", err)
	}
	got, err = io.ReadAll(dup)
	if err != nil {
		t.Fatalf("read duplicate after original close: %v", err)
	}
	if string(got) != "duplicate me" {
		t.Fatalf("read duplicate after original close = %q, want %q", got, "duplicate me")
	}
}
