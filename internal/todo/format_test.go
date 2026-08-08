package todo

import (
	"reflect"
	"testing"
)

// The _format pipeline (§6.2): numbering → filters → sort → colors, plus
// the summary helper. Expected outputs mirror the real todo.sh (t1300,
// t1330, t1380).

// tasksFromTexts builds Tasks with consecutive 1-based line numbers.
func tasksFromTexts(texts ...string) []Task {
	tasks := make([]Task, len(texts))
	for i, text := range texts {
		tasks[i] = Task{LineNumber: i + 1, Text: text}
	}
	return tasks
}

func TestFormatNumberingAndSort(t *testing.T) {
	tasks := tasksFromTexts(
		"ccc xxx this line should be third.",
		"aaa zzz this line should be first.",
		"bbb yyy this line should be second.",
	)
	lines, shown, total, err := Format(tasks, nil, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"2 aaa zzz this line should be first.",
		"3 bbb yyy this line should be second.",
		"1 ccc xxx this line should be third.",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format = %v, want %v", lines, want)
	}
	if shown != 3 || total != 3 {
		t.Errorf("Format counts = %d/%d, want 3/3", shown, total)
	}
}

// Numbers are zero-padded to the width of the file's line count: 10 lines
// → 2 digits (t1300 'check line number padding').
func TestFormatPadding(t *testing.T) {
	texts := make([]string, 10)
	for i := range texts {
		texts[i] = "task"
	}
	lines, _, _, err := Format(tasksFromTexts(texts...), nil, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(lines, []string{
		"01 task", "02 task", "03 task", "04 task", "05 task",
		"06 task", "07 task", "08 task", "09 task", "10 task",
	}) {
		t.Errorf("Format = %v", lines)
	}
}

// Blank lines (and lines that are only digits/spaces, which the numbering
// sed cannot distinguish from blanks) are dropped, keeping their real line
// numbers (t1300 'check that blank lines are ignored').
func TestFormatDropsBlanks(t *testing.T) {
	tasks := tasksFromTexts("hex00 this is one line", "", "hex02 this is another line", "42")
	lines, shown, total, err := Format(tasks, nil, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1 hex00 this is one line", "3 hex02 this is another line"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format = %v, want %v", lines, want)
	}
	if shown != 2 || total != 2 {
		t.Errorf("Format counts = %d/%d, want 2/2", shown, total)
	}
}

// The numbering width is derived from the highest line number, so a file
// with trailing blank lines still pads correctly (getPadding counts all
// lines).
func TestFormatWidthFromLineCount(t *testing.T) {
	tasks := tasksFromTexts("one", "two")
	lines, _, _, err := Format(tasks, nil, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "1 one" {
		t.Errorf("Format = %v, want width 1", lines)
	}
	lines, _, _, err = Format(tasks, nil, FormatOptions{Width: 3})
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "001 one" {
		t.Errorf("Format with Width 3 = %v", lines)
	}
}

// Non-sequential line numbers come from the tasks themselves (listall's
// zeroed numbers are just LineNumber 0).
func TestFormatKeepsTaskNumbers(t *testing.T) {
	tasks := []Task{
		{LineNumber: 5, Text: "stop"},
		{LineNumber: 1, Text: "aaa"},
		{LineNumber: 0, Text: "x done"},
	}
	lines, _, _, err := Format(tasks, nil, FormatOptions{Width: 1})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1 aaa", "5 stop", "0 x done"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format = %v, want %v", lines, want)
	}
}

func TestFormatFilters(t *testing.T) {
	tasks := tasksFromTexts(
		"ccc xxx this line should be third.",
		"aaa zzz this line should be first.",
		"bbb yyy this line should be second.",
	)
	lines, shown, total, err := Format(tasks, []string{"second"}, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"3 bbb yyy this line should be second."}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format = %v, want %v", lines, want)
	}
	if shown != 1 || total != 3 {
		t.Errorf("Format counts = %d/%d, want 1/3", shown, total)
	}
	lines, _, _, err = Format(tasks, []string{"-second"}, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"2 aaa zzz this line should be first.", "1 ccc xxx this line should be third."}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format(-second) = %v, want %v", lines, want)
	}
}

func TestFormatEmpty(t *testing.T) {
	lines, shown, total, err := Format(nil, nil, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 || shown != 0 || total != 0 {
		t.Errorf("Format on empty = %v, %d/%d", lines, shown, total)
	}
}

// t1330 'default highlighting': priority lines take pri_<letter>, letters
// without their own color fall back to pri_x, done lines take color_done.
func TestFormatPriorityColors(t *testing.T) {
	tasks := tasksFromTexts(
		"(A) @con01 +prj01 -- Some project 01 task, pri A",
		"(D) @con02 +prj02 -- Some project 02 task, pri D",
		"@con01 +prj01 -- no priority",
		"x 2009-02-13 remove1",
	)
	lines, _, _, err := Format(tasks, nil, FormatOptions{Colors: testColors{defaultPalette}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"\x1b[1;33m1 (A) @con01 +prj01 -- Some project 01 task, pri A\x1b[0m",
		"\x1b[1;37m2 (D) @con02 +prj02 -- Some project 02 task, pri D\x1b[0m",
		"3 @con01 +prj01 -- no priority",
		"\x1b[0;37m4 x 2009-02-13 remove1\x1b[0m",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format = %q, want %q", lines, want)
	}
}

// t1380: the number word takes color_number; a valid date takes color_date;
// key:value words take color_meta. Each colored word resets to DEFAULT and
// re-applies the line color (empty here).
func TestFormatWordColors(t *testing.T) {
	tasks := tasksFromTexts(
		"2018-11-11 task with date",
		"task with metadata due:2018-12-31",
		"task without date and without metadata",
	)
	lines, _, _, err := Format(tasks, nil, FormatOptions{Colors: testColors{wordPalette}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"\x1b[0;34m1\x1b[0m \x1b[0;31m2018-11-11\x1b[0m task with date",
		"\x1b[0;34m2\x1b[0m task with metadata \x1b[0;32mdue:2018-12-31\x1b[0m",
		"\x1b[0;34m3\x1b[0m task without date and without metadata",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format = %q, want %q", lines, want)
	}
}

// The awk date regex validates month and day ranges: 2018-13-45 is not a
// color_date word.
func TestFormatInvalidDateNotColored(t *testing.T) {
	tasks := tasksFromTexts("2018-13-45 not a date")
	lines, _, _, err := Format(tasks, nil, FormatOptions{Colors: testColors{wordPalette}})
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "\x1b[0;34m1\x1b[0m 2018-13-45 not a date" {
		t.Errorf("Format = %q", lines[0])
	}
}

// On a colored line the reset after a colored word is DEFAULT followed by
// the line color, and the line ends with DEFAULT (awk prj_end = DEFAULT +
// clr).
func TestFormatWordResetOnColoredLine(t *testing.T) {
	tasks := tasksFromTexts("(A) due:2018-12-31 task")
	lines, _, _, err := Format(tasks, nil, FormatOptions{Colors: testColors{wordPalette}})
	if err != nil {
		t.Fatal(err)
	}
	want := "\x1b[1;33m\x1b[0;34m1\x1b[0m\x1b[1;33m (A) \x1b[0;32mdue:2018-12-31\x1b[0m\x1b[1;33m task\x1b[0m"
	if lines[0] != want {
		t.Errorf("Format = %q, want %q", lines[0], want)
	}
}

// Plain mode (nil Colors) emits no escape sequences at all.
func TestFormatPlain(t *testing.T) {
	tasks := tasksFromTexts("(A) @con01 +prj01 -- task")
	lines, _, _, err := Format(tasks, nil, FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "1 (A) @con01 +prj01 -- task" {
		t.Errorf("Format plain = %q", lines[0])
	}
}

// -@ hides contexts, -+ hides projects: the sigil word and its preceding
// space are removed from the colorized line (t1300 suppression test).
func TestFormatHideSigils(t *testing.T) {
	tasks := tasksFromTexts("(A) @con01 +prj01 -- Some project 01 task, pri A")
	opts := FormatOptions{Colors: testColors{defaultPalette}, HideContexts: true}
	lines, _, _, err := Format(tasks, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "\x1b[1;33m1 (A) +prj01 -- Some project 01 task, pri A\x1b[0m" {
		t.Errorf("HideContexts = %q", lines[0])
	}
	opts = FormatOptions{Colors: testColors{defaultPalette}, HideProjects: true}
	lines, _, _, err = Format(tasks, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "\x1b[1;33m1 (A) @con01 -- Some project 01 task, pri A\x1b[0m" {
		t.Errorf("HideProjects = %q", lines[0])
	}
	opts = FormatOptions{Colors: testColors{defaultPalette}, HideContexts: true, HideProjects: true}
	lines, _, _, err = Format(tasks, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if lines[0] != "\x1b[1;33m1 (A) -- Some project 01 task, pri A\x1b[0m" {
		t.Errorf("Hide both = %q", lines[0])
	}
}

// -P removes the line-initial priority label after the line color was
// chosen (awkStep's splice); mid-text and done-line "(X) " labels stay,
// verified against the real todo.sh (t1300).
func TestFormatHidePriority(t *testing.T) {
	tasks := tasksFromTexts(
		"(A) @con01 +prj01 -- Some project 01 task, pri A",
		"(A) foo (B) bar",
		"x (A) foo",
	)
	lines, _, _, err := Format(tasks, nil, FormatOptions{Colors: testColors{defaultPalette}, HidePriority: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"\x1b[1;33m1 @con01 +prj01 -- Some project 01 task, pri A\x1b[0m",
		"\x1b[1;33m2 foo (B) bar\x1b[0m",
		"\x1b[0;37m3 x (A) foo\x1b[0m",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("HidePriority = %q, want %q", lines, want)
	}
}

// Summary emits the `--` separator and the counts line, exactly like
// _list's tail (t0001: 0 of 0 still shown).
func TestSummary(t *testing.T) {
	if got := Summary("TODO", 1, 3); !reflect.DeepEqual(got, []string{"--", "TODO: 1 of 3 tasks shown"}) {
		t.Errorf("Summary = %v", got)
	}
	if got := Summary("DONE", 0, 0); !reflect.DeepEqual(got, []string{"--", "DONE: 0 of 0 tasks shown"}) {
		t.Errorf("Summary = %v", got)
	}
}

// Prefix is getPrefix: basename without extension, uppercased (t1020:
// GARDEN, t1350: TODO/DONE).
func TestPrefix(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"/home/user/todo.txt", "TODO"},
		{"/home/user/done.txt", "DONE"},
		{"garden.txt", "GARDEN"},
		{"foo.bar.txt", "FOO.BAR"},
		{"noext", "NOEXT"},
	} {
		if got := Prefix(tc.path); got != tc.want {
			t.Errorf("Prefix(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// PostFilter runs between the term filters and the sort — todo.sh's
// post_filter_command, which listall uses to renumber done.txt tasks to 0
// (t1350).
func TestFormatPostFilter(t *testing.T) {
	tasks := tasksFromTexts("aaa", "x done one", "x done two")
	opts := FormatOptions{
		Width: 1,
		PostFilter: func(lines []string) ([]string, error) {
			out := make([]string, len(lines))
			for i, line := range lines {
				if line[0] > '1' {
					out[i] = "0" + line[1:]
				} else {
					out[i] = line
				}
			}
			return out, nil
		},
	}
	lines, _, _, err := Format(tasks, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1 aaa", "0 x done one", "0 x done two"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("Format with PostFilter = %v, want %v", lines, want)
	}
}

// An invalid term surfaces as a filter error (grep's "unmatched" case).
func TestFormatBadTerm(t *testing.T) {
	if _, _, _, err := Format(tasksFromTexts("one"), []string{`\(`}, FormatOptions{}); err == nil {
		t.Error("Format with bad term: want error")
	}
}
