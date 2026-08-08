package todo

import (
	"reflect"
	"strings"
	"testing"
)

// TestSigilWords pins todo.sh's listWordsWithSigil with the default sigil
// patterns (t1310/t1320): terms filter the raw lines, then every
// space-separated run starting with the sigil (length ≥ 2) is listed,
// sorted byte-wise and deduplicated. Runs that merely contain the sigil
// are not listed; tabs are part of a run, not separators.
func TestSigilWords(t *testing.T) {
	lines := []string{
		"(B) +math (@school or @home) integrate @x and @y",
		"(C) say thanks @GinaTrapani w:@OtherContributors",
		"@con01 @con01 @con02",
		"@tab\tpart",
	}
	got, err := SigilWords(lines, '@', nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"@GinaTrapani", "@con01", "@con02", "@home)", "@tab\tpart", "@x", "@y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SigilWords = %v, want %v", got, want)
	}
}

// SigilWords filters the lines with the terms before extracting, like
// listWordsWithSigil's filtercommand step.
func TestSigilWordsTerms(t *testing.T) {
	lines := []string{
		"(B) smell the uppercase Roses +roses @outside +shared",
		"(C) notice the sunflowers +sunflowers @garden +shared +landscape",
		"stop",
	}
	got, err := SigilWords(lines, '+', []string{"@garden"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"+landscape", "+shared", "+sunflowers"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SigilWords with terms = %v, want %v", got, want)
	}
}

// TestCompilePriorityClass pins the GNU grep bracket-expression rules of
// listpri's post-filter class (t1250): letters are literals, X-Y is an
// ascending range, and malformed ranges fail with grep's diagnostic.
func TestCompilePriorityClass(t *testing.T) {
	for _, tc := range []struct {
		pri  string
		want string // sorted members of the class
	}{
		{"A", "A"},
		{"CX", "CX"},
		{"A-C", "ABC"},
		{"ABR-Y", "ABRSTUVWXY"},
		{"C-Z", "CDEFGHIJKLMNOPQRSTUVWXYZ"},
	} {
		class, err := CompilePriorityClass(tc.pri)
		if err != nil {
			t.Errorf("CompilePriorityClass(%q): %v", tc.pri, err)
			continue
		}
		var got strings.Builder
		for _, letter := range []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			if class(letter) {
				got.WriteByte(letter)
			}
		}
		if got.String() != tc.want {
			t.Errorf("CompilePriorityClass(%q) matches %q, want %q", tc.pri, got.String(), tc.want)
		}
	}
	// Reversed and repeated ranges fail the whole grep, like GNU grep.
	for _, pri := range []string{"Z-A", "A-B-C", "A--Z"} {
		if _, err := CompilePriorityClass(pri); err == nil || err.Error() != "grep: invalid character range" {
			t.Errorf("CompilePriorityClass(%q) = %v, want the grep diagnostic", pri, err)
		}
	}
}

// TestListpriArg pins the detection grep of todo.sh's listpri: a single
// letter, a letter-dash-letter, or an uppercase sequence of letters and
// dashes, uppercased on match. "A-" does not match and stays a filter term.
func TestListpriArg(t *testing.T) {
	for _, arg := range []string{"A", "a", "A-C", "c-Z", "CX", "ABR-Y", "Z-A", "A--Z"} {
		if got, ok := ListpriArg(arg); !ok || got != strings.ToUpper(arg) {
			t.Errorf("ListpriArg(%q) = %q, %v; want %q, true", arg, got, ok, strings.ToUpper(arg))
		}
	}
	for _, arg := range []string{"", "A-", "-A", "1", "A C", "-"} {
		if _, ok := ListpriArg(arg); ok {
			t.Errorf("ListpriArg(%q) matched, want no match", arg)
		}
	}
}

// TestListallPostFilter pins listall's renumbering awk (t1350): numbers
// above total become 0, all numbers stay right-aligned in width, and the
// field rebuild collapses whitespace runs.
func TestListallPostFilter(t *testing.T) {
	in := []string{
		" 1 smell the   roses",
		" 2 x 2011-01-01 done task",
		" 3 x 2010-01-01 old task",
	}
	got, err := ListallPostFilter(2, 1)(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1 smell the roses", "2 x 2011-01-01 done task", "0 x 2010-01-01 old task"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListallPostFilter = %v, want %v", got, want)
	}
}

// ListableCount excludes blank and digit-only lines, the lines the
// numbering sed drops (they determine TOTALTASKS).
func TestListableCount(t *testing.T) {
	tasks := []Task{
		{LineNumber: 1, Text: "a task"},
		{LineNumber: 2, Text: ""},
		{LineNumber: 3, Text: "   "},
		{LineNumber: 4, Text: "123"},
		{LineNumber: 5, Text: "12x"},
	}
	if got := ListableCount(tasks); got != 2 {
		t.Errorf("ListableCount = %d, want 2", got)
	}
}
