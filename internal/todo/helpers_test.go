package todo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedNow is the clock the mutation tests pin: the todo.sh test suite runs
// under TZ=UTC on 2009-02-13 (t1010, t1500), and report tests use the
// 2009-02-13T04:40:00 timestamp (t1950).
var fixedNow = time.Date(2009, 2, 13, 4, 40, 0, 0, time.UTC)

// newTestStore builds a Store on a fresh temp dir with the todo/done/report
// files created (§6.5) and todo written to the todo file when non-empty.
// The default knobs mirror todo.cfg: preserve line numbers on, sentence
// delimiters ",.:;".
func newTestStore(t *testing.T, todo string) *Store {
	t.Helper()
	dir := t.TempDir()
	s := &Store{
		Dir:                 dir,
		TodoFile:            filepath.Join(dir, "todo.txt"),
		DoneFile:            filepath.Join(dir, "done.txt"),
		ReportFile:          filepath.Join(dir, "report.txt"),
		PreserveLineNumbers: true,
		SentenceDelimiters:  ",.:;",
	}
	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if todo != "" {
		writeFile(t, s.TodoFile, todo)
	}
	return s
}

// writeFile replaces the content of path.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readFile returns the content of path.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// testColors is a Colorer with a named palette of real ANSI strings, so
// the format tests pin the exact bytes of the colorized output. The
// palettes mirror todo.cfg: by default only pri_*/color_done are set; the
// t1380-style tests add number/date/meta colors.
type testColors struct{ palette map[string]string }

var (
	// defaultPalette: todo.cfg's defaults (YELLOW/GREEN/LIGHT_BLUE/WHITE
	// for pri, LIGHT_GREY for done; project/context/date/number/meta are
	// NONE).
	defaultPalette = map[string]string{
		"pri_a":      "\x1b[1;33m",
		"pri_b":      "\x1b[0;32m",
		"pri_c":      "\x1b[1;34m",
		"pri_x":      "\x1b[1;37m",
		"color_done": "\x1b[0;37m",
		"default":    "\x1b[0m",
	}

	// wordPalette: t1380's highlighting of numbers, dates, and metadata.
	wordPalette = map[string]string{
		"pri_a":        "\x1b[1;33m",
		"color_number": "\x1b[0;34m",
		"color_date":   "\x1b[0;31m",
		"color_meta":   "\x1b[0;32m",
		"default":      "\x1b[0m",
	}
)

func (c testColors) Color(role string) string {
	return c.palette[strings.ToLower(role)]
}

func (c testColors) PriorityColor(letter byte) string {
	if clr, ok := c.palette["pri_"+strings.ToLower(string(letter))]; ok {
		return clr
	}
	return c.palette["pri_x"]
}
