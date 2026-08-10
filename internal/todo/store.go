package todo

// The file store (§6.5): TODO_DIR and the todo/done/report files are
// created on demand, and the mutations read and write the files line-wise
// with real 1-based line numbers. File content is modeled as lines plus a
// trailing-newline flag so that the sed semantics of todo.sh survive byte
// for byte: a missing final EOL is preserved unless the last line is
// removed or a line is appended.

import (
	"fmt"
	"os"
	"strings"
)

// Store owns the todo/done/report files and the file-level knobs of the
// mutating actions. It is a pure file store: prompts, usage errors, and
// message formatting belong to the CLI layer (Tasks 6-8).
type Store struct {
	// Dir is TODO_DIR; Ensure creates it and the three files on demand.
	Dir        string
	TodoFile   string
	DoneFile   string
	ReportFile string

	// PreserveLineNumbers mirrors TODOTXT_PRESERVE_LINE_NUMBERS: del and
	// move leave a blank line behind instead of removing the line.
	PreserveLineNumbers bool

	// SentenceDelimiters mirrors SENTENCE_DELIMITERS: append inserts no
	// space before text starting with one of these characters.
	SentenceDelimiters string
}

// Ensure creates TODO_DIR (mkdir -p) and the todo/done/report files when
// missing, like todo.sh's startup sanity checks (§6.5). Existing files are
// left untouched.
func (s *Store) Ensure() error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		// The exact todo.sh dieWithHelp text, contract (§6.5).
		return fmt.Errorf("Fatal Error: %s is not a directory", s.Dir) //nolint:staticcheck,revive
	}
	for _, path := range []string{s.TodoFile, s.DoneFile, s.ReportFile} {
		if err := ensureFile(path); err != nil {
			return err
		}
	}
	return nil
}

// ensureFile creates path as an empty file when it does not exist,
// mirroring todo.sh's `: > "$TODO_FILE"` (which also fails on directories).
func ensureFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

// ReadTasks reads path into Tasks with their real 1-based line numbers.
// Blank lines are included: they determine the numbering width and the
// archive/report counts; the _format pipeline drops them itself.
func (s *Store) ReadTasks(path string) ([]Task, error) {
	lines, _, err := readLines(path)
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, len(lines))
	for i, line := range lines {
		tasks[i] = Task{LineNumber: i + 1, Text: line}
	}
	return tasks, nil
}

// ReadAllTasks returns the tasks of todo.txt followed by done.txt,
// numbered continuously 1..N across both files — the shape of
// `cat "$TODO_FILE" "$DONE_FILE"` that listall formats (§6.3).
func (s *Store) ReadAllTasks() ([]Task, error) {
	var out []Task
	n := 0
	for _, path := range []string{s.TodoFile, s.DoneFile} {
		tasks, err := s.ReadTasks(path)
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			n++
			out = append(out, Task{LineNumber: n, Text: t.Text})
		}
	}
	return out, nil
}

// readLines splits path into its lines. finalNL reports whether the file
// ends with a newline; an empty file yields no lines.
func readLines(path string) (lines []string, finalNL bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	finalNL = data[len(data)-1] == '\n'
	lines = strings.Split(string(data), "\n")
	if finalNL {
		lines = lines[:len(lines)-1]
	}
	return lines, finalNL, nil
}

// writeLines writes the lines joined by newlines, terminating the last one
// iff finalNL — the exact file shape sed leaves behind.
func writeLines(path string, lines []string, finalNL bool) error {
	return os.WriteFile(path, linesData(lines, finalNL), 0o644)
}

func linesData(lines []string, finalNL bool) []byte {
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	// A single blank line still gets its newline: sed leaves "\n" when the
	// last remaining line is blanked (e.g. preserve-mode del of the only
	// line), so guard on the line count, not the builder length.
	if finalNL && len(lines) > 0 {
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// appendTo appends add to path with the semantics of `>>`: when the file
// lacks a final newline, the first appended line joins the last existing
// line (grep/echo do not insert a separator).
func appendTo(path string, add []string) error {
	lines, finalNL, err := readLines(path)
	if err != nil {
		return err
	}
	if !finalNL && len(lines) > 0 && len(add) > 0 {
		lines[len(lines)-1] += add[0]
		add = add[1:]
	}
	return writeLines(path, append(lines, add...), true)
}

// CountLines returns the number of lines in path, todo.sh's `sed -n '$ ='`
// (blank lines count).
func (s *Store) CountLines(path string) (int, error) {
	lines, _, err := readLines(path)
	if err != nil {
		return 0, err
	}
	return len(lines), nil
}

// isRegular mirrors todo.sh's `[ -f path ]`.
func isRegular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
