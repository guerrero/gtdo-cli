package cli_test

// Ports of the tests that a txtar script cannot express: t2000-multiline.sh
// (arguments with embedded newlines — testscript command lines are single
// lines) and the two sessions that need a file without a trailing newline
// (t1000 'add to file without EOL' and t1850 'move to destination without
// EOL' — txtar file sections always gain a final newline).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guerrero/gtdo/internal/cli"
)

// runGTDO runs cli.Run in a fresh HOME with the harness environment (pinned
// clock, hermetic config) and returns stdout, stderr, and the exit code.
func runGTDO(t *testing.T, dir, stdin string, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("GTDO_CONFIG", filepath.Join(dir, "gtdo-config.json"))
	t.Setenv("GTDO_TEST_NOW", "2009-02-13T04:40:00Z")
	var out, errw strings.Builder
	code := cli.Run(args, strings.NewReader(stdin), &out, &errw)
	return out.String(), errw.String(), code
}

func mustRun(t *testing.T, dir, stdin, wantOut string, wantCode int, args ...string) {
	t.Helper()
	out, errw, code := runGTDO(t, dir, stdin, args...)
	if out != wantOut {
		t.Errorf("%v:\nstdout = %q\nwant    %q", args, out, wantOut)
	}
	if errw != "" {
		t.Errorf("%v: unexpected stderr %q", args, errw)
	}
	if code != wantCode {
		t.Errorf("%v: exit code = %d, want %d", args, code, wantCode)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestMultilineSquash is t2000-multiline.sh: replace, add, append, and
// prepend squash embedded newlines into spaces; addm adds each line as
// its own task.
func TestMultilineSquash(t *testing.T) {
	dir := t.TempDir()
	todo := filepath.Join(dir, "todo.txt")

	mustRun(t, dir, "", "1 smell the cheese\nTODO: 1 added.\n", 0, "add", "smell the cheese")

	mustRun(t, dir, "", "1 smell the cheese\nTODO: Replaced task with:\n1 eat apples eat oranges drink milk\n", 0,
		"replace", "1", "eat apples\neat oranges\ndrink milk")

	writeFile(t, todo, "")
	mustRun(t, dir, "", "1 smell the cheese\nTODO: 1 added.\n", 0, "add", "smell the cheese")
	mustRun(t, dir, "", "2 eat apples eat oranges drink milk\nTODO: 2 added.\n", 0,
		"add", "eat apples\neat oranges\ndrink milk")

	writeFile(t, todo, "")
	mustRun(t, dir, "", "1 smell the cheese\nTODO: 1 added.\n", 0, "add", "smell the cheese")
	mustRun(t, dir, "", "1 smell the cheese eat apples eat oranges drink milk\n", 0,
		"append", "1", "eat apples\neat oranges\ndrink milk")

	writeFile(t, todo, "")
	mustRun(t, dir, "", "1 smell the cheese\nTODO: 1 added.\n", 0, "add", "smell the cheese")
	mustRun(t, dir, "", "1 eat apples eat oranges drink milk smell the cheese\n", 0,
		"prepend", "1", "eat apples\neat oranges\ndrink milk")

	// actual multiline add: each line becomes its own task
	mustRun(t, dir, "", "2 eat apples\nTODO: 2 added.\n3 eat oranges\nTODO: 3 added.\n4 drink milk\nTODO: 4 added.\n", 0,
		"addm", "eat apples\neat oranges\ndrink milk")
}

// TestAddToFileWithoutEOL is t1000 'add to file without EOL': the missing
// final newline is fixed before the task is appended (fixMissingEndOfLine).
func TestAddToFileWithoutEOL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "todo.txt"), "this is a first task without newline")

	mustRun(t, dir, "", "2 a second task\nTODO: 2 added.\n", 0, "add", "a second task")

	if got := readFile(t, filepath.Join(dir, "todo.txt")); got != "this is a first task without newline\na second task\n" {
		t.Errorf("todo.txt = %q, want the newline fixed before the append", got)
	}
}

// TestMoveDestWithoutEOL is t1850 'move to destination without EOL': the
// destination gets its missing final newline before the moved task lands.
func TestMoveDestWithoutEOL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "todo.txt"), "this is a first task without newline")
	writeFile(t, filepath.Join(dir, "done.txt"), "x 2009-02-13 make the coffee +wakeup\nx 2009-02-13 smell the coffee +wakeup\n")

	mustRun(t, dir, "", "2 x 2009-02-13 smell the coffee +wakeup\nDONE: 2 moved to 2 in TODO.\n", 0,
		"-f", "move", "2", "todo.txt", "done.txt")

	if got := readFile(t, filepath.Join(dir, "todo.txt")); got != "this is a first task without newline\nx 2009-02-13 smell the coffee +wakeup\n" {
		t.Errorf("todo.txt = %q, want the moved task on its own line", got)
	}
	// PreserveLineNumbers is the default: the moved line leaves a blank
	// behind in done.txt.
	if got := readFile(t, filepath.Join(dir, "done.txt")); got != "x 2009-02-13 make the coffee +wakeup\n\n" {
		t.Errorf("done.txt = %q, want the moved line blanked", got)
	}
}
