package todo

import (
	"reflect"
	"testing"
)

// Filter test cases pin todo.sh's filtercommand semantics (§6.2.2): terms
// AND'ed with grep -i (basic regex), "-TERM" excludes, "\|" ORs, empty terms
// match everything. The numbered lines below mirror what the _format
// pipeline feeds the filter (t1300 ls fixtures, verified against the real
// todo.sh).

func TestFilterLines(t *testing.T) {
	lines := []string{
		"1 ccc xxx this line should be third.",
		"2 aaa zzz this line should be first.",
		"3 bbb yyy this line should be second.",
	}
	cases := []struct {
		name  string
		terms []string
		want  []string
	}{
		{"single term", []string{"second"}, []string{lines[2]}},
		{"partial word", []string{"should be f"}, []string{lines[1]}},
		{"leading space", []string{" zzz"}, []string{lines[1]}},
		{"char class", []string{"ir[ds]"}, []string{lines[0], lines[1]}},
		{"dot star", []string{"f.*t"}, []string{lines[1]}},
		{"terms AND'ed", []string{"ir[ds]", "xxx"}, []string{lines[0]}},
		{"exclusion", []string{"-second"}, []string{lines[0], lines[1]}},
		{"exclusion with space", []string{"-should be f"}, []string{lines[0], lines[2]}},
		{"exclusion leading space", []string{"- zzz"}, []string{lines[0], lines[2]}},
		{"case-insensitive", []string{"FIRST."}, []string{lines[1]}},
		{"empty term matches everything", []string{""}, lines},
		{"bare dash excludes everything", []string{"-"}, []string{}},
	}
	for _, c := range cases {
		got, err := FilterLines(lines, c.terms)
		if err != nil {
			t.Errorf("%s: FilterLines(%v) error: %v", c.name, c.terms, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: FilterLines(%v) = %v, want %v", c.name, c.terms, got, c.want)
		}
	}
}

// The filter runs on numbered lines, so the line number participates in
// matching (verified against the real todo.sh: "ls 1" on a 12-line file
// shows lines 1, 10, 11 and 12).
func TestFilterLinesNumberParticipation(t *testing.T) {
	lines := []string{
		"01 line one", "02 line two", "03 line three", "04 line four",
		"05 line five", "06 line six", "07 line seven", "08 line eight",
		"09 line nine", "10 line ten", "11 line eleven", "12 line twelve",
	}
	got, err := FilterLines(lines, []string{"1"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"01 line one", "10 line ten", "11 line eleven", "12 line twelve"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterLines(lines, [1]) = %v, want %v", got, want)
	}
}

// Terms may contain characters special to the shell; they must be treated
// literally (t1300 'filtering of special characters').
func TestFilterLinesSpecialCharacters(t *testing.T) {
	lines := []string{
		"1 earn some pennies",
		"2 earn some $$",
		"3 earn some \"money\"",
		"4 get money from O'Brian",
		"5 just get   money!",
	}
	cases := []struct {
		term string
		want []string
	}{
		{"$$", []string{lines[1]}},
		{"\"money\"", []string{lines[2]}},
		{"O'Brian", []string{lines[3]}},
		{"get   money", []string{lines[4]}}, // multiple spaces are significant
	}
	for _, c := range cases {
		got, err := FilterLines(lines, []string{c.term})
		if err != nil {
			t.Errorf("term %q: %v", c.term, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("FilterLines(lines, [%q]) = %v, want %v", c.term, got, c.want)
		}
	}
}

// \| inside a term is an OR of alternates (GNU grep BRE alternation, which
// RE2 spells "|").
func TestFilterLinesOr(t *testing.T) {
	lines := []string{
		"1 foo bar",
		"2 baz qux",
		"3 nothing",
	}
	got, err := FilterLines(lines, []string{"bar\\|qux"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{lines[0], lines[1]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterLines(lines, [bar\\|qux]) = %v, want %v", got, want)
	}
}

// Unescaped BRE metacharacters are literals: "(A)" matches only lines that
// literally contain "(A)".
func TestFilterLinesLiteralParens(t *testing.T) {
	lines := []string{
		"1 (A) prioritized task",
		"2 a task",
	}
	got, err := FilterLines(lines, []string{"(A)"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{lines[0]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterLines(lines, [(A)]) = %v, want %v", got, want)
	}
}

// A plus or question mark unescaped is a literal in BRE, not a quantifier.
func TestFilterLinesLiteralQuantifiers(t *testing.T) {
	lines := []string{
		"1 a+b",
		"2 ab",
		"3 a?b",
	}
	for _, term := range []string{"a+b", "a?b"} {
		got, err := FilterLines(lines, []string{term})
		if err != nil {
			t.Errorf("term %q: %v", term, err)
			continue
		}
		want := []string{lines[0]}
		if term == "a?b" {
			want = []string{lines[2]}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("FilterLines(lines, [%q]) = %v, want %v", term, got, want)
		}
	}
}

// Escaped BRE operators keep their operator meaning: "\+", "\?" and "\{m,n\}"
// are quantifiers in both grep BRE and RE2.
func TestFilterLinesBREOperators(t *testing.T) {
	lines := []string{
		"1 foobar",
		"2 foo bar",
	}
	cases := []struct {
		term string
		want []string
	}{
		{"foo\\+bar", []string{lines[0]}},
		{"foo\\?bar", []string{lines[0]}},
		{"foo\\{1,2\\} bar", []string{lines[1]}},
		{"foo\\(bar\\)", []string{lines[0]}},
	}
	for _, c := range cases {
		got, err := FilterLines(lines, []string{c.term})
		if err != nil {
			t.Errorf("term %q: %v", c.term, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("FilterLines(lines, [%q]) = %v, want %v", c.term, got, c.want)
		}
	}
}

// A malformed term is an error, like grep reporting an unmatched bracket.
func TestFilterLinesInvalidTerm(t *testing.T) {
	if _, err := FilterLines([]string{"1 foo"}, []string{"["}); err == nil {
		t.Error("FilterLines with term \"[\" succeeded, want an error")
	}
}

func TestMatchLine(t *testing.T) {
	ok, err := MatchLine("2 aaa zzz this line should be first.", "zzz")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("MatchLine should have matched")
	}
	ok, err = MatchLine("2 aaa zzz", "no match")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("MatchLine should not have matched")
	}
}
