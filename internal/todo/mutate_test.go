package todo

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Mutations (§6.3), each pinned against the real todo.sh: file state and
// return values. The CLI layer (Tasks 6-8) owns prompts, usage errors, and
// message formatting; these tests assert the file+line operations.

func TestAddAppends(t *testing.T) {
	s := newTestStore(t, "")
	line, text, err := s.Add("notice the daisies", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != 1 || text != "notice the daisies" {
		t.Errorf("Add = (%d, %q), want (1, %q)", line, text, "notice the daisies")
	}
	line, text, err = s.Add("smell the roses", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != 2 || text != "smell the roses" {
		t.Errorf("second Add = (%d, %q), want (2, smell the roses)", line, text)
	}
	if got := readFile(t, s.TodoFile); got != "notice the daisies\nsmell the roses\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

// Spaces are preserved verbatim: cleaninput only maps CR/LF to spaces, it
// does not trim (verified against the real todo.sh).
func TestAddPreservesSpaces(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("  padded  ", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "  padded  " {
		t.Errorf("Add text = %q, want %q", text, "  padded  ")
	}
}

// CR and LF become spaces, so a multi-line argument lands as one task
// (todo.sh cleaninput; t1000 'add with CR', t2000).
func TestAddSquashesCRLF(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("smell the\rCarriage Return", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "smell the Carriage Return" {
		t.Errorf("Add = %q, want %q", text, "smell the Carriage Return")
	}
}

// A lowercase priority is uppercased on add (todo.sh uppercasePriority;
// t1010).
func TestAddUppercasesPriority(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("(b) notice the daisies", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "(B) notice the daisies" {
		t.Errorf("Add = %q, want %q", text, "(B) notice the daisies")
	}
}

func TestAddDateOnAdd(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("notice the daisies", true, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "2009-02-13 notice the daisies" {
		t.Errorf("Add = %q, want %q", text, "2009-02-13 notice the daisies")
	}
}

// The date goes after an existing priority: `(A) 2009-02-13 task` (t1010).
func TestAddDateAfterPriority(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("(A) notice the daisies", true, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "(A) 2009-02-13 notice the daisies" {
		t.Errorf("Add = %q, want %q", text, "(A) 2009-02-13 notice the daisies")
	}
}

func TestAddPriorityOnAdd(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("take out the trash", false, "A", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "(A) take out the trash" {
		t.Errorf("Add = %q, want %q", text, "(A) take out the trash")
	}
}

// priority_on_add is skipped when the task already carries a priority.
func TestAddPriorityOnAddKeepsExisting(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("(B) take out the trash", false, "A", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "(B) take out the trash" {
		t.Errorf("Add = %q, want %q", text, "(B) take out the trash")
	}
}

// date_on_add and priority_on_add combine: priority first, then date.
func TestAddDateAndPriorityOnAdd(t *testing.T) {
	s := newTestStore(t, "")
	_, text, err := s.Add("take out the trash", true, "A", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if text != "(A) 2009-02-13 take out the trash" {
		t.Errorf("Add = %q, want %q", text, "(A) 2009-02-13 take out the trash")
	}
}

// fixMissingEndOfLine: a file without a trailing newline gets one before
// the append (t1000 'add to file without EOL').
func TestAddToFileWithoutEOL(t *testing.T) {
	s := newTestStore(t, "")
	writeFile(t, s.TodoFile, "this is a first task without newline")
	line, _, err := s.Add("a second task", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != 2 {
		t.Errorf("Add line = %d, want 2", line)
	}
	if got := readFile(t, s.TodoFile); got != "this is a first task without newline\na second task\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

// Adding an empty argument appends a blank line (t0001 territory: the line
// count grows; the listing drops it).
func TestAddEmpty(t *testing.T) {
	s := newTestStore(t, "one\n")
	line, text, err := s.Add("", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != 2 || text != "" {
		t.Errorf("Add = (%d, %q), want (2, )", line, text)
	}
}

func TestAddmSplitsLines(t *testing.T) {
	s := newTestStore(t, "smell the cheese\n")
	res, err := s.Addm("eat apples\neat oranges\ndrink milk", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	want := []AddResult{
		{LineNumber: 2, Text: "eat apples"},
		{LineNumber: 3, Text: "eat oranges"},
		{LineNumber: 4, Text: "drink milk"},
	}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("Addm = %+v, want %+v", res, want)
	}
	if got := readFile(t, s.TodoFile); got != "smell the cheese\neat apples\neat oranges\ndrink milk\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

// Bash word splitting drops empty fields, so empty lines add nothing.
func TestAddmDropsEmptyLines(t *testing.T) {
	s := newTestStore(t, "")
	res, err := s.Addm("a\n\nb", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Errorf("Addm = %+v, want 2 results", res)
	}
}

func TestAddmEmpty(t *testing.T) {
	s := newTestStore(t, "")
	res, err := s.Addm("", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Errorf("Addm(\"\") = %+v, want none", res)
	}
}

func TestAddto(t *testing.T) {
	s := newTestStore(t, "")
	garden := filepath.Join(s.Dir, "garden.txt")
	writeFile(t, garden, "")
	line, text, err := s.Addto("garden.txt", "notice the daisies", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != 1 || text != "notice the daisies" {
		t.Errorf("Addto = (%d, %q), want (1, notice the daisies)", line, text)
	}
	if got := readFile(t, garden); got != "notice the daisies\n" {
		t.Errorf("garden.txt = %q", got)
	}
}

func TestAddtoMissingDest(t *testing.T) {
	s := newTestStore(t, "")
	_, _, err := s.Addto("garden.txt", "notice the daisies", false, "", fixedNow)
	if err == nil {
		t.Fatal("Addto to missing dest: want error")
	}
	want := "TODO: Destination file " + filepath.Join(s.Dir, "garden.txt") + " does not exist."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAppendAddsSpace(t *testing.T) {
	s := newTestStore(t, "notice the daisies\n")
	text, err := s.Append(1, "smell the roses")
	if err != nil {
		t.Fatal(err)
	}
	if text != "notice the daisies smell the roses" {
		t.Errorf("Append = %q, want %q", text, "notice the daisies smell the roses")
	}
}

// Sentence delimiters (SENTENCE_DELIMITERS, default ",.:;"): no space is
// inserted when the appended text starts with one (t1600).
func TestAppendSentenceDelimiter(t *testing.T) {
	s := newTestStore(t, "notice the daisies\n")
	for _, tc := range []struct{ text, want string }{
		{", lilies and roses", "notice the daisies, lilies and roses"},
		{"; see the wasps", "notice the daisies, lilies and roses; see the wasps"},
		{"& bees", "notice the daisies, lilies and roses; see the wasps & bees"},
	} {
		text, err := s.Append(1, tc.text)
		if err != nil {
			t.Fatal(err)
		}
		if text != tc.want {
			t.Errorf("Append(%q) = %q, want %q", tc.text, text, tc.want)
		}
	}
}

func TestAppendCustomDelimiters(t *testing.T) {
	s := newTestStore(t, "notice the daisies\n")
	s.SentenceDelimiters = "*,.:;&"
	text, err := s.Append(1, "&beans")
	if err != nil {
		t.Fatal(err)
	}
	if text != "notice the daisies&beans" {
		t.Errorf("Append(&beans) = %q, want no space", text)
	}
	text, err = s.Append(1, "%foo")
	if err != nil {
		t.Fatal(err)
	}
	if text != "notice the daisies&beans %foo" {
		t.Errorf("Append(%%foo) = %q, want space", text)
	}
}

// The sed-escaped characters land literally in the file (t1600 'append with
// symbols'): in Go the text is inserted verbatim, no escaping needed.
func TestAppendLiteralSpecials(t *testing.T) {
	s := newTestStore(t, "smell the cows\n")
	text, err := s.Append(1, "~@#$%^&*()-_=+[{]}|;:',<.>/?")
	if err != nil {
		t.Fatal(err)
	}
	if text != "smell the cows ~@#$%^&*()-_=+[{]}|;:',<.>/?" {
		t.Errorf("Append = %q", text)
	}
	got := readFile(t, s.TodoFile)
	if !strings.Contains(got, "&") || !strings.Contains(got, "|") {
		t.Errorf("todo.txt = %q, want literal & and |", got)
	}
}

func TestAppendBackslash(t *testing.T) {
	s := newTestStore(t, "grow some corn\n")
	text, err := s.Append(1, "`!\\\"")
	if err != nil {
		t.Fatal(err)
	}
	if text != "grow some corn `!\\\"" {
		t.Errorf("Append = %q, want %q", text, "grow some corn `!\\\"")
	}
}

func TestAppendNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, err := s.Append(9, "hej!"); err == nil || err.Error() != "TODO: No task 9." {
		t.Errorf("Append(9) error = %v, want TODO: No task 9.", err)
	}
}

// append does not touch the file's end-of-line state: a file without a
// trailing newline stays that way (sed preserves it).
func TestAppendNoEOL(t *testing.T) {
	s := newTestStore(t, "")
	writeFile(t, s.TodoFile, "jump on hay")
	text, err := s.Append(1, "and notice the   three   spaces")
	if err != nil {
		t.Fatal(err)
	}
	if text != "jump on hay and notice the   three   spaces" {
		t.Errorf("Append = %q", text)
	}
	if got := readFile(t, s.TodoFile); got != "jump on hay and notice the   three   spaces" {
		t.Errorf("todo.txt = %q, want no trailing newline", got)
	}
}

func TestPrependNoSpace(t *testing.T) {
	s := newTestStore(t, "notice the sunflowers\n")
	text, err := s.Prepend(1, "test")
	if err != nil {
		t.Fatal(err)
	}
	if text != "test notice the sunflowers" {
		t.Errorf("Prepend = %q, want %q", text, "test notice the sunflowers")
	}
}

// The existing priority is preserved (t1400).
func TestPrependKeepsPriority(t *testing.T) {
	s := newTestStore(t, "(B) smell the uppercase Roses +flowers @outside\n")
	text, err := s.Prepend(1, "test")
	if err != nil {
		t.Fatal(err)
	}
	if text != "(B) test smell the uppercase Roses +flowers @outside" {
		t.Errorf("Prepend = %q", text)
	}
}

// The existing date prefix is preserved (t1400 'prepend handling prepended
// date on add').
func TestPrependKeepsDate(t *testing.T) {
	s := newTestStore(t, "2009-02-13 new task\n")
	text, err := s.Prepend(1, "this is just a")
	if err != nil {
		t.Fatal(err)
	}
	if text != "2009-02-13 this is just a new task" {
		t.Errorf("Prepend = %q", text)
	}
}

func TestPrependKeepsPriorityAndDate(t *testing.T) {
	s := newTestStore(t, "(A) 2009-02-13 new task\n")
	text, err := s.Prepend(1, "this is just a")
	if err != nil {
		t.Fatal(err)
	}
	if text != "(A) 2009-02-13 this is just a new task" {
		t.Errorf("Prepend = %q", text)
	}
}

// A date in the prepended text is not extracted: prepend keeps both dates
// (t1400 'prepend with prepended date keeps both').
func TestPrependKeepsBothDates(t *testing.T) {
	s := newTestStore(t, "2009-02-13 new task\n")
	text, err := s.Prepend(1, "2010-07-04 this is just a")
	if err != nil {
		t.Fatal(err)
	}
	if text != "2009-02-13 2010-07-04 this is just a new task" {
		t.Errorf("Prepend = %q", text)
	}
}

func TestPrependLiteralSpecials(t *testing.T) {
	s := newTestStore(t, "stop\n")
	text, err := s.Prepend(1, "no running & jumping now")
	if err != nil {
		t.Fatal(err)
	}
	if text != "no running & jumping now stop" {
		t.Errorf("Prepend = %q", text)
	}
}

func TestPrependNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, err := s.Prepend(9, "x"); err == nil || err.Error() != "TODO: No task 9." {
		t.Errorf("Prepend(9) error = %v, want TODO: No task 9.", err)
	}
}

func TestReplace(t *testing.T) {
	s := newTestStore(t, "notice the daisies\n")
	oldText, newText, err := s.Replace(1, "smell the cows")
	if err != nil {
		t.Fatal(err)
	}
	if oldText != "notice the daisies" || newText != "smell the cows" {
		t.Errorf("Replace = (%q, %q)", oldText, newText)
	}
	if got := readFile(t, s.TodoFile); got != "smell the cows\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

func TestReplaceKeepsPriority(t *testing.T) {
	s := newTestStore(t, "(A) collect the eggs\n")
	_, newText, err := s.Replace(1, "collect the bread")
	if err != nil {
		t.Fatal(err)
	}
	if newText != "(A) collect the bread" {
		t.Errorf("Replace = %q, want %q", newText, "(A) collect the bread")
	}
}

// A date in the replacement text replaces the existing date, the priority
// stays (t1100 'replace with prepended date replaces existing date').
func TestReplaceInputDateReplaces(t *testing.T) {
	s := newTestStore(t, "(A) 2009-02-13 this is just a new one\n")
	_, newText, err := s.Replace(1, "2010-07-04 this also has a new date")
	if err != nil {
		t.Fatal(err)
	}
	if newText != "(A) 2010-07-04 this also has a new date" {
		t.Errorf("Replace = %q", newText)
	}
}

// A priority in the replacement text replaces the existing priority, the
// date stays (t1100).
func TestReplaceInputPriorityReplaces(t *testing.T) {
	s := newTestStore(t, "(A) 2009-02-13 this is just a new one\n")
	_, newText, err := s.Replace(1, "(B) this also has a new priority")
	if err != nil {
		t.Fatal(err)
	}
	if newText != "(B) 2009-02-13 this also has a new priority" {
		t.Errorf("Replace = %q", newText)
	}
}

func TestReplaceInputPriorityAndDateReplace(t *testing.T) {
	s := newTestStore(t, "(A) 2009-02-13 this is just a new one\n")
	_, newText, err := s.Replace(1, "(B) 2010-07-04 this also has a new prio+date")
	if err != nil {
		t.Fatal(err)
	}
	if newText != "(B) 2010-07-04 this also has a new prio+date" {
		t.Errorf("Replace = %q", newText)
	}
}

func TestReplaceLiteralSpecials(t *testing.T) {
	s := newTestStore(t, "jump on hay\n")
	_, newText, err := s.Replace(1, "thrash the hay & thrash the wheat")
	if err != nil {
		t.Fatal(err)
	}
	if newText != "thrash the hay & thrash the wheat" {
		t.Errorf("Replace = %q", newText)
	}
}

func TestReplaceNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, _, err := s.Replace(9, "x"); err == nil || err.Error() != "TODO: No task 9." {
		t.Errorf("Replace(9) error = %v, want TODO: No task 9.", err)
	}
}

func TestPriSets(t *testing.T) {
	s := newTestStore(t, "smell the uppercase Roses +flowers @outside\n")
	res, err := s.Pri(1, 'B')
	if err != nil {
		t.Fatal(err)
	}
	want := PriResult{LineNumber: 1, NewText: "(B) smell the uppercase Roses +flowers @outside", OldPri: 0, NewPri: 'B'}
	if res != want {
		t.Errorf("Pri = %+v, want %+v", res, want)
	}
}

func TestPriReprioritizes(t *testing.T) {
	s := newTestStore(t, "(C) notice the sunflowers\n")
	res, err := s.Pri(1, 'A')
	if err != nil {
		t.Fatal(err)
	}
	if res.NewText != "(A) notice the sunflowers" || res.OldPri != 'C' {
		t.Errorf("Pri = %+v", res)
	}
}

// The priority is re-prioritized even when the old letter is lowercase:
// `s/^(.) //` matches any character in the parens (t1200 'pri 2 a').
func TestPriLowercaseOld(t *testing.T) {
	s := newTestStore(t, "(c) foo\n")
	res, err := s.Pri(1, 'A')
	if err != nil {
		t.Fatal(err)
	}
	if res.NewText != "(A) foo" || res.OldPri != 'c' {
		t.Errorf("Pri = %+v", res)
	}
}

func TestPriAlready(t *testing.T) {
	s := newTestStore(t, "(A) foo\n")
	res, err := s.Pri(1, 'A')
	if err != nil {
		t.Fatal(err)
	}
	if res.NewText != "(A) foo" || res.OldPri != 'A' || res.NewPri != 'A' {
		t.Errorf("Pri = %+v, want unchanged (A) foo with OldPri A", res)
	}
	if got := readFile(t, s.TodoFile); got != "(A) foo\n" {
		t.Errorf("todo.txt = %q, want untouched", got)
	}
}

// The date survives re-prioritization.
func TestPriKeepsDate(t *testing.T) {
	s := newTestStore(t, "(C) 2009-02-13 foo\n")
	res, err := s.Pri(1, 'A')
	if err != nil {
		t.Fatal(err)
	}
	if res.NewText != "(A) 2009-02-13 foo" {
		t.Errorf("Pri = %q, want %q", res.NewText, "(A) 2009-02-13 foo")
	}
}

func TestPriNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, err := s.Pri(9, 'A'); err == nil || err.Error() != "TODO: No task 9." {
		t.Errorf("Pri(9) error = %v, want TODO: No task 9.", err)
	}
}

func TestDepri(t *testing.T) {
	s := newTestStore(t, "(B) smell the uppercase Roses +flowers @outside\n(A) notice the sunflowers\n(C) stop\n")
	res, err := s.Depri([]int{3, 2})
	if err != nil {
		t.Fatal(err)
	}
	want := []DepriResult{
		{LineNumber: 3, NewText: "stop", Prioritized: true},
		{LineNumber: 2, NewText: "notice the sunflowers", Prioritized: true},
	}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("Depri = %+v, want %+v", res, want)
	}
	if got := readFile(t, s.TodoFile); got != "(B) smell the uppercase Roses +flowers @outside\nnotice the sunflowers\nstop\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

// `s/^(.) //` strips any single character in the parens, lowercase included.
func TestDepriLowercase(t *testing.T) {
	s := newTestStore(t, "(b) foo\n")
	res, err := s.Depri([]int{1})
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].Prioritized || res[0].NewText != "foo" {
		t.Errorf("Depri = %+v", res)
	}
}

func TestDepriNotPrioritized(t *testing.T) {
	s := newTestStore(t, "stop\n")
	res, err := s.Depri([]int{1})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Prioritized || res[0].NewText != "stop" {
		t.Errorf("Depri = %+v, want not prioritized", res)
	}
	if got := readFile(t, s.TodoFile); got != "stop\n" {
		t.Errorf("todo.txt = %q, want untouched", got)
	}
}

func TestDepriNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, err := s.Depri([]int{42}); err == nil || err.Error() != "TODO: No task 42." {
		t.Errorf("Depri(42) error = %v, want TODO: No task 42.", err)
	}
}

// Items before the failing one stay processed, like todo.sh's sequential
// sed calls.
func TestDepriPartialResults(t *testing.T) {
	s := newTestStore(t, "(A) one\nstop\n")
	res, err := s.Depri([]int{1, 42})
	if err == nil || err.Error() != "TODO: No task 42." {
		t.Fatalf("Depri error = %v, want TODO: No task 42.", err)
	}
	if len(res) != 1 || res[0].LineNumber != 1 {
		t.Errorf("Depri results = %+v, want item 1 only", res)
	}
}

func TestDoMarksDone(t *testing.T) {
	s := newTestStore(t, "smell the uppercase Roses +flowers @outside\n")
	res, err := s.Do([]int{1}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	want := DoResult{LineNumber: 1, NewText: "x 2009-02-13 smell the uppercase Roses +flowers @outside", AlreadyDone: false}
	if !reflect.DeepEqual(res, []DoResult{want}) {
		t.Errorf("Do = %+v, want %+v", res, want)
	}
}

// todo.sh removes the priority when marking done — `sed "${item}s/^(.) //"`
// before prepending "x DATE " (the plan's "preserving priority" note is
// wrong; verified against the real todo.sh).
func TestDoRemovesPriority(t *testing.T) {
	s := newTestStore(t, "(A) foo bar\n")
	res, err := s.Do([]int{1}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].NewText != "x 2009-02-13 foo bar" {
		t.Errorf("Do = %q, want %q", res[0].NewText, "x 2009-02-13 foo bar")
	}
}

// A leading date is not a priority and survives.
func TestDoKeepsDate(t *testing.T) {
	s := newTestStore(t, "2009-02-13 foo\n")
	res, err := s.Do([]int{1}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].NewText != "x 2009-02-13 2009-02-13 foo" {
		t.Errorf("Do = %q, want %q", res[0].NewText, "x 2009-02-13 2009-02-13 foo")
	}
}

func TestDoMultiple(t *testing.T) {
	s := newTestStore(t, "remove1\nremove2\nremove3\nremove4\n")
	res, err := s.Do([]int{4, 3}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0].LineNumber != 4 || res[1].LineNumber != 3 {
		t.Errorf("Do = %+v", res)
	}
	if got := readFile(t, s.TodoFile); got != "remove1\nremove2\nx 2009-02-13 remove3\nx 2009-02-13 remove4\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

func TestDoAlreadyDone(t *testing.T) {
	s := newTestStore(t, "x 2009-02-13 stop\n")
	res, err := s.Do([]int{1}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].AlreadyDone || res[0].NewText != "x 2009-02-13 stop" {
		t.Errorf("Do = %+v, want AlreadyDone", res)
	}
	if got := readFile(t, s.TodoFile); got != "x 2009-02-13 stop\n" {
		t.Errorf("todo.txt = %q, want untouched", got)
	}
}

func TestDoNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, err := s.Do([]int{42}, fixedNow); err == nil || err.Error() != "TODO: No task 42." {
		t.Errorf("Do(42) error = %v, want TODO: No task 42.", err)
	}
}

func TestDelPreserve(t *testing.T) {
	s := newTestStore(t, "(B) smell the uppercase Roses +flowers @outside\n(A) notice the sunflowers\nstop\n")
	oldText, err := s.Del(1)
	if err != nil {
		t.Fatal(err)
	}
	if oldText != "(B) smell the uppercase Roses +flowers @outside" {
		t.Errorf("Del = %q", oldText)
	}
	// The line stays as a blank (line numbers preserved); the next add
	// lands on line 4 (t1800 'del preserving line numbers').
	if got := readFile(t, s.TodoFile); got != "\n(A) notice the sunflowers\nstop\n" {
		t.Errorf("todo.txt = %q", got)
	}
	line, _, err := s.Add("A new task", false, "", fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != 4 {
		t.Errorf("Add after del = line %d, want 4", line)
	}
}

func TestDelNotPreserve(t *testing.T) {
	s := newTestStore(t, "one\ntwo\nthree\n")
	s.PreserveLineNumbers = false
	if _, err := s.Del(2); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, s.TodoFile); got != "one\nthree\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

// Non-preserve deletion also compacts pre-existing blank lines, like
// sed -e '/./!d' applied to the whole file.
func TestDelNotPreserveCompactsBlanks(t *testing.T) {
	s := newTestStore(t, "one\n\ntwo\n")
	s.PreserveLineNumbers = false
	if _, err := s.Del(1); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, s.TodoFile); got != "two\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

// Deleting the unterminated last line leaves the file ending with the
// previous line's newline (sed behavior).
func TestDelLastLineNoEOL(t *testing.T) {
	s := newTestStore(t, "")
	writeFile(t, s.TodoFile, "a\nb")
	s.PreserveLineNumbers = false
	if _, err := s.Del(2); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, s.TodoFile); got != "a\n" {
		t.Errorf("todo.txt = %q, want %q", got, "a\n")
	}
}

func TestDelNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	if _, err := s.Del(42); err == nil || err.Error() != "TODO: No task 42." {
		t.Errorf("Del(42) error = %v, want TODO: No task 42.", err)
	}
}

// The five sed rules of del TERM (t1800 'basic del TERM').
func TestDelTerm(t *testing.T) {
	s := newTestStore(t, "(B) smell the uppercase Roses +flowers @outside\n(A) notice the sunflowers\n(C) stop\n")
	for _, tc := range []struct{ term, want string }{
		{"uppercase", "(B) smell the Roses +flowers @outside"},
		{"the Roses", "(B) smell +flowers @outside"},
		{"m", "(B) sell +flowers @outside"},
		{"@outside", "(B) sell +flowers"},
		{"sell", "(B) +flowers"},
	} {
		oldText, newText, err := s.DelTerm(1, tc.term)
		if err != nil {
			t.Fatalf("DelTerm(%q): %v", tc.term, err)
		}
		if newText != tc.want {
			t.Errorf("DelTerm(%q) = %q, want %q (old %q)", tc.term, newText, tc.want, oldText)
		}
	}
}

// The first rule keeps the priority and its following spaces: `(B)  foo`
// minus `foo` leaves `(B) `, not `(B)  `.
func TestDelTermKeepsPriority(t *testing.T) {
	s := newTestStore(t, "(B)  foo bar  \n")
	_, newText, err := s.DelTerm(1, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if newText != "(B) bar  " {
		t.Errorf("DelTerm = %q, want %q", newText, "(B) bar  ")
	}
}

func TestDelTermNotFound(t *testing.T) {
	s := newTestStore(t, "(B) smell the uppercase Roses +flowers @outside\n")
	_, _, err := s.DelTerm(1, "dung")
	if err == nil || err.Error() != "TODO: 'dung' not found; no removal done." {
		t.Errorf("DelTerm error = %v, want TODO: 'dung' not found; no removal done.", err)
	}
	if got := readFile(t, s.TodoFile); got != "(B) smell the uppercase Roses +flowers @outside\n" {
		t.Errorf("todo.txt = %q, want untouched", got)
	}
}

func TestMove(t *testing.T) {
	s := newTestStore(t, "(B) smell the uppercase Roses +flowers @outside\n(A) notice the sunflowers\n")
	writeFile(t, s.DoneFile, "x 2009-02-13 make the coffee +wakeup\nx 2009-02-13 smell the coffee +wakeup\n")
	oldText, destNum, err := s.Move(1, s.DoneFile, s.TodoFile)
	if err != nil {
		t.Fatal(err)
	}
	if oldText != "(B) smell the uppercase Roses +flowers @outside" || destNum != 3 {
		t.Errorf("Move = (%q, %d), want (..., 3)", oldText, destNum)
	}
	if got := readFile(t, s.TodoFile); got != "\n(A) notice the sunflowers\n" {
		t.Errorf("todo.txt = %q, want blank line 1", got)
	}
	if got := readFile(t, s.DoneFile); got != "x 2009-02-13 make the coffee +wakeup\nx 2009-02-13 smell the coffee +wakeup\n(B) smell the uppercase Roses +flowers @outside\n" {
		t.Errorf("done.txt = %q", got)
	}
}

// Moving from a non-default source: the "No task" error carries the source
// prefix (DONE for done.txt).
func TestMoveFromDone(t *testing.T) {
	s := newTestStore(t, "(A) notice the sunflowers\n")
	writeFile(t, s.DoneFile, "x 2009-02-13 make the coffee +wakeup\nx 2009-02-13 smell the coffee +wakeup\n")
	oldText, destNum, err := s.Move(2, s.TodoFile, s.DoneFile)
	if err != nil {
		t.Fatal(err)
	}
	if oldText != "x 2009-02-13 smell the coffee +wakeup" || destNum != 2 {
		t.Errorf("Move = (%q, %d), want (x 2009-02-13 smell the coffee +wakeup, 2)", oldText, destNum)
	}
}

func TestMoveNoTask(t *testing.T) {
	s := newTestStore(t, "one\n")
	writeFile(t, s.DoneFile, "")
	if _, _, err := s.Move(42, s.DoneFile, s.TodoFile); err == nil || err.Error() != "TODO: No task 42." {
		t.Errorf("Move(42) error = %v, want TODO: No task 42.", err)
	}
	if _, _, err := s.Move(42, s.TodoFile, s.DoneFile); err == nil || err.Error() != "DONE: No task 42." {
		t.Errorf("Move from done error = %v, want DONE: No task 42.", err)
	}
}

func TestMoveDestMissing(t *testing.T) {
	s := newTestStore(t, "one\n")
	missing := filepath.Join(s.Dir, "missing.txt")
	if _, _, err := s.Move(1, missing, s.TodoFile); err == nil || err.Error() != "TODO: Destination file "+missing+" does not exist." {
		t.Errorf("Move error = %v", err)
	}
}

func TestMoveSrcMissing(t *testing.T) {
	s := newTestStore(t, "one\n")
	missing := filepath.Join(s.Dir, "missing.txt")
	writeFile(t, s.DoneFile, "")
	if _, _, err := s.Move(1, s.DoneFile, missing); err == nil || err.Error() != "TODO: Source file "+missing+" does not exist." {
		t.Errorf("Move error = %v", err)
	}
}

func TestMoveNotPreserve(t *testing.T) {
	s := newTestStore(t, "(A) notice the sunflowers\n")
	writeFile(t, s.DoneFile, "")
	s.PreserveLineNumbers = false
	if _, _, err := s.Move(1, s.DoneFile, s.TodoFile); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, s.TodoFile); got != "" {
		t.Errorf("todo.txt = %q, want empty", got)
	}
}

// The destination gets an end-of-line before the append (t1850 'move to
// destination without EOL').
func TestMoveDestNoEOL(t *testing.T) {
	s := newTestStore(t, "this is a first task without newline")
	writeFile(t, s.DoneFile, "x 2009-02-13 make the coffee +wakeup\nx 2009-02-13 smell the coffee +wakeup\n")
	oldText, destNum, err := s.Move(2, s.TodoFile, s.DoneFile)
	if err != nil {
		t.Fatal(err)
	}
	if oldText != "x 2009-02-13 smell the coffee +wakeup" || destNum != 2 {
		t.Errorf("Move = (%q, %d)", oldText, destNum)
	}
	if got := readFile(t, s.TodoFile); got != "this is a first task without newline\nx 2009-02-13 smell the coffee +wakeup\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

func TestArchive(t *testing.T) {
	s := newTestStore(t, "one\ntwo\nthree\none\nx done\nfour\n")
	moved, err := s.Archive()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(moved, []string{"x done"}) {
		t.Errorf("Archive = %v, want [x done]", moved)
	}
	if got := readFile(t, s.TodoFile); got != "one\ntwo\nthree\none\nfour\n" {
		t.Errorf("todo.txt = %q", got)
	}
	if got := readFile(t, s.DoneFile); got != "x done\n" {
		t.Errorf("done.txt = %q", got)
	}
}

// Archive always defragments blank lines, even when there is nothing done
// (t1900 sequence).
func TestArchiveRemovesBlanks(t *testing.T) {
	s := newTestStore(t, "a\n\nb\n")
	moved, err := s.Archive()
	if err != nil {
		t.Fatal(err)
	}
	if moved != nil {
		t.Errorf("Archive = %v, want none", moved)
	}
	if got := readFile(t, s.TodoFile); got != "a\nb\n" {
		t.Errorf("todo.txt = %q", got)
	}
}

func TestArchiveNoDone(t *testing.T) {
	s := newTestStore(t, "one\ntwo\n")
	moved, err := s.Archive()
	if err != nil {
		t.Fatal(err)
	}
	if moved != nil {
		t.Errorf("Archive = %v, want none", moved)
	}
	if got := readFile(t, s.DoneFile); got != "" {
		t.Errorf("done.txt = %q, want empty", got)
	}
}

// The x lines append after the existing done content.
func TestArchiveAppendsToDone(t *testing.T) {
	s := newTestStore(t, "x one\nkeep\n")
	writeFile(t, s.DoneFile, "old done\n")
	if _, err := s.Archive(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, s.DoneFile); got != "old done\nx one\n" {
		t.Errorf("done.txt = %q", got)
	}
}

// grep appends raw lines: a done file without a trailing newline joins its
// last line with the first archived task (verified against the real
// `grep >> `).
func TestArchiveDoneNoEOL(t *testing.T) {
	s := newTestStore(t, "x one\n")
	writeFile(t, s.DoneFile, "old done")
	if _, err := s.Archive(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, s.DoneFile); got != "old donex one\n" {
		t.Errorf("done.txt = %q, want %q", got, "old donex one\n")
	}
}

func TestReportWrites(t *testing.T) {
	s := newTestStore(t, "one\ntwo\nthree\nfour\nfive\n")
	writeFile(t, s.DoneFile, "x a\nx b\n")
	line, updated, err := s.Report(fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != "2009-02-13T04:40:00 5 2" || !updated {
		t.Errorf("Report = (%q, %v), want (2009-02-13T04:40:00 5 2, true)", line, updated)
	}
	if got := readFile(t, s.ReportFile); got != "2009-02-13T04:40:00 5 2\n" {
		t.Errorf("report.txt = %q", got)
	}
}

func TestReportUpToDate(t *testing.T) {
	s := newTestStore(t, "one\n")
	writeFile(t, s.DoneFile, "")
	first, _, err := s.Report(fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	line, updated, err := s.Report(fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != first || updated {
		t.Errorf("second Report = (%q, %v), want unchanged, not updated", line, updated)
	}
	if got := readFile(t, s.ReportFile); got != first+"\n" {
		t.Errorf("report.txt = %q, want one line", got)
	}
}

func TestReportCountsChanged(t *testing.T) {
	s := newTestStore(t, "one\n")
	writeFile(t, s.DoneFile, "")
	if _, _, err := s.Report(fixedNow); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Add("two", false, "", fixedNow); err != nil {
		t.Fatal(err)
	}
	line, updated, err := s.Report(fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if line != "2009-02-13T04:40:00 2 0" || !updated {
		t.Errorf("Report = (%q, %v), want (2009-02-13T04:40:00 2 0, true)", line, updated)
	}
	if got := readFile(t, s.ReportFile); got != "2009-02-13T04:40:00 1 0\n2009-02-13T04:40:00 2 0\n" {
		t.Errorf("report.txt = %q", got)
	}
}
