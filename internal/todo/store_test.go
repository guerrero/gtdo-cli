package todo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The file store (§6.5): Ensure creates TODO_DIR and the todo/done/report
// files on demand; ReadTasks returns the raw lines with their real 1-based
// line numbers, blank lines included.

func TestEnsureCreatesDirAndFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested", "todo")
	s := &Store{
		Dir:        sub,
		TodoFile:   filepath.Join(sub, "todo.txt"),
		DoneFile:   filepath.Join(sub, "done.txt"),
		ReportFile: filepath.Join(sub, "report.txt"),
	}
	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	for _, f := range []string{s.TodoFile, s.DoneFile, s.ReportFile} {
		if b, err := os.ReadFile(f); err != nil || len(b) != 0 {
			t.Errorf("%s: want empty file, got %q, %v", f, b, err)
		}
	}
}

func TestEnsureKeepsExistingFiles(t *testing.T) {
	s := newTestStore(t, "one\ntwo\n")
	writeFile(t, s.DoneFile, "x done\n")
	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got := readFile(t, s.TodoFile); got != "one\ntwo\n" {
		t.Errorf("todo.txt = %q, want unchanged", got)
	}
	if got := readFile(t, s.DoneFile); got != "x done\n" {
		t.Errorf("done.txt = %q, want unchanged", got)
	}
}

func TestReadTasks(t *testing.T) {
	s := newTestStore(t, "first\n\nthird\n")
	tasks, err := s.ReadTasks(s.TodoFile)
	if err != nil {
		t.Fatal(err)
	}
	want := []Task{
		{LineNumber: 1, Text: "first"},
		{LineNumber: 2, Text: ""},
		{LineNumber: 3, Text: "third"},
	}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("ReadTasks = %+v, want %+v", tasks, want)
	}
}

// A file without a trailing newline still yields its lines; the last line
// simply has no terminator (the mutations preserve that state, like sed).
func TestReadTasksNoFinalNewline(t *testing.T) {
	s := newTestStore(t, "")
	writeFile(t, s.TodoFile, "no-eol-task")
	tasks, err := s.ReadTasks(s.TodoFile)
	if err != nil {
		t.Fatal(err)
	}
	want := []Task{{LineNumber: 1, Text: "no-eol-task"}}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("ReadTasks = %+v, want %+v", tasks, want)
	}
}

func TestReadTasksEmptyFile(t *testing.T) {
	s := newTestStore(t, "")
	tasks, err := s.ReadTasks(s.TodoFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 0 {
		t.Errorf("ReadTasks on empty file = %+v, want none", tasks)
	}
}
