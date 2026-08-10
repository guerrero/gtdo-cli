package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/guerrero/gtdo/internal/config"
)

func TestCollectAddCandidatesUnionsTaskFiles(t *testing.T) {
	dir := t.TempDir()
	todoPath := filepath.Join(dir, "todo.txt")
	donePath := filepath.Join(dir, "done.txt")
	reportPath := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(todoPath, []byte("one @home +gtdo due:today\ntwo @work +personal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(donePath, []byte("done @home +archive due:yesterday status:done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, []byte("report @decoy +decoy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := collectAddCandidates(&config.Config{TodoFile: todoPath, DoneFile: donePath, ReportFile: reportPath})
	want := addCandidates{
		Contexts: []string{"@home", "@work"},
		Projects: []string{"+archive", "+gtdo", "+personal"},
		Metadata: []metadataCandidate{
			{Key: "due", Values: []string{"today", "yesterday"}},
			{Key: "status", Values: []string{"done"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectAddCandidates() = %#v, want %#v", got, want)
	}
}

func TestCollectAddCandidatesDeduplicatesPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	if err := os.WriteFile(path, []byte("task @home +gtdo due:today\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := collectAddCandidatesFromPaths([]string{path, path})
	want := addCandidates{
		Contexts: []string{"@home"},
		Projects: []string{"+gtdo"},
		Metadata: []metadataCandidate{{Key: "due", Values: []string{"today"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectAddCandidatesFromPaths() = %#v, want %#v", got, want)
	}
}

func TestCollectAddCandidatesIgnoresMissingAndEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	emptyPath := filepath.Join(dir, "empty.txt")
	missingPath := filepath.Join(dir, "missing.txt")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got := collectAddCandidatesFromPaths([]string{emptyPath, missingPath})
	if !reflect.DeepEqual(got, addCandidates{}) {
		t.Errorf("collectAddCandidatesFromPaths() = %#v, want no candidates", got)
	}
	if _, err := os.Stat(missingPath); !os.IsNotExist(err) {
		t.Errorf("collector created missing path %q", missingPath)
	}
}

func TestCollectAddCandidatesIgnoresMalformedMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	text := "bad-key:value :empty key: 1bad:ok Key:UPPER key:one:two valid:ok\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	got := collectAddCandidatesFromPaths([]string{path})
	want := addCandidates{
		Metadata: []metadataCandidate{
			{Key: "1bad", Values: []string{"ok"}},
			{Key: "Key", Values: []string{"UPPER"}},
			{Key: "key", Values: []string{"one:two"}},
			{Key: "valid", Values: []string{"ok"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectAddCandidatesFromPaths() = %#v, want %#v", got, want)
	}
}

func TestCollectAddCandidatesSortsByteWiseAndCaseSensitively(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	text := "@z @A @a @Z +z +A +a +Z order:z order:A order:a order:Z\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	got := collectAddCandidatesFromPaths([]string{path})
	want := addCandidates{
		Contexts: []string{"@A", "@Z", "@a", "@z"},
		Projects: []string{"+A", "+Z", "+a", "+z"},
		Metadata: []metadataCandidate{{Key: "order", Values: []string{"A", "Z", "a", "z"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectAddCandidatesFromPaths() = %#v, want %#v", got, want)
	}
}
