package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
)

// todoFileConfig returns a config whose TODO_FILE is path; the completion
// functions read nothing else.
func todoFileConfig(path string) *config.Config {
	return &config.Config{TodoFile: path}
}

// runCompletions invokes fn the way cobra's __complete does and returns
// the candidates and directive.
func runCompletions(fn cobra.CompletionFunc, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return fn(&cobra.Command{}, args, toComplete)
}

// TestTaskNumbers pins the number candidates of the NR-taking actions:
// the real line numbers of the listable lines (blank lines and lines of
// only digits and spaces are dropped, exactly like the `list` numbering),
// filtered by the typed prefix. An empty prefix offers every number; a
// missing file offers none.
func TestTaskNumbers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.txt")
	// Line 3 is blank, line 5 is only digits: neither gets a number.
	text := "(A) one @ctx\n(B) two +proj\n\nfour\n12345\nx 2009-02-13 done\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		args   []string
		toComp string
		want   []string
	}{
		{name: "empty prefix lists every listable line", toComp: "", want: []string{"1", "2", "4", "6"}},
		{name: "prefix filters", toComp: "2", want: []string{"2"}},
		{name: "no match yields nothing", toComp: "9", want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, directive := runCompletions(everyArgNumbers(todoFileConfig(path)), tc.args, tc.toComp)
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
			}
			if !equalCompletions(got, tc.want) {
				t.Errorf("completions = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTaskNumbersMissingFile pins the never-fail rule: a missing or
// unreadable TODO_FILE yields no completions, and the command is never
// asked to create anything (completions read the file directly).
func TestTaskNumbersMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.txt")
	got, directive := runCompletions(everyArgNumbers(todoFileConfig(path)), nil, "")
	if len(got) != 0 {
		t.Errorf("completions = %q, want none", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("completion created a file")
	}
}

// TestNumberPositions pins which argument positions offer numbers: the
// first argument of append/del/move/prepend/replace, every argument of
// do/depri, and the even positions of pri (its NR PRIORITY pairs).
func TestNumberPositions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		fn   cobra.CompletionFunc
		args []string
		want int // expected number of candidates
	}{
		{name: "append first arg", fn: firstArgNumbers(todoFileConfig(path)), want: 2},
		{name: "append second arg", fn: firstArgNumbers(todoFileConfig(path)), args: []string{"1"}, want: 0},
		{name: "do every arg", fn: everyArgNumbers(todoFileConfig(path)), args: []string{"1"}, want: 2},
		{name: "pri even positions", fn: evenArgNumbers(todoFileConfig(path)), args: []string{"1", "A"}, want: 2},
		{name: "pri odd positions", fn: evenArgNumbers(todoFileConfig(path)), args: []string{"1"}, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := runCompletions(tc.fn, tc.args, "")
			if len(got) != tc.want {
				t.Errorf("got %d completions %q, want %d", len(got), got, tc.want)
			}
		})
	}
}

// TestTermCompletions pins the list/listall candidates: the @contexts and
// +projects of TODO_FILE, exactly the words listcon/listproj list, with
// the typed sigil picking the kind and the prefix filtering the words.
func TestTermCompletions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.txt")
	// "@phone" appears twice, "@travel" once, "+Family" twice; the
	// digit-only line and the embedded sigils (w:@Other) are not words.
	text := "(A) Call Mom @phone +Family\n(B) Plan @phone @travel +Family\n12345\nw:@Other (+school)\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	fn := termCompletions(todoFileConfig(path))

	cases := []struct {
		name   string
		toComp string
		want   []string
	}{
		{name: "contexts", toComp: "@", want: []string{"@phone", "@travel"}},
		{name: "projects", toComp: "+", want: []string{"+Family"}},
		{name: "prefix filters", toComp: "@tr", want: []string{"@travel"}},
		{name: "no sigil offers both", toComp: "", want: []string{"@phone", "@travel", "+Family"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, directive := runCompletions(fn, nil, tc.toComp)
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
			}
			if !equalCompletions(got, tc.want) {
				t.Errorf("completions = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTermCompletionsMissingFile pins the never-fail rule for terms too.
func TestTermCompletionsMissingFile(t *testing.T) {
	got, directive := runCompletions(termCompletions(todoFileConfig(filepath.Join(t.TempDir(), "absent.txt"))), nil, "@")
	if len(got) != 0 || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("got %q with directive %v, want none with ShellCompDirectiveNoFileComp", got, directive)
	}
}

func equalCompletions(got []cobra.Completion, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if string(got[i]) != want[i] {
			return false
		}
	}
	return true
}
