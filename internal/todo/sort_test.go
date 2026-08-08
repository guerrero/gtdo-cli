package todo

import (
	"reflect"
	"testing"
)

// SortLines pins the _format sort step (§6.2.3): `LC_COLLATE=C sort -f -k2`
// on numbered lines — case-insensitive by the task text, with ties broken by
// the line number (GNU sort's last-resort comparison of the full line).
// Orderings below were verified against the real todo.sh / BSD sort.

func TestSortLinesCaseInsensitive(t *testing.T) {
	lines := []string{"1 ZZZ line", "2 aaa line", "3 Mmm line"}
	want := []string{"2 aaa line", "3 Mmm line", "1 ZZZ line"}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines = %v, want %v", lines, want)
	}
}

// The sort key is the full task text including any priority or date prefix:
// "(A) zebra" sorts before "(B) apple" (verified against the real todo.sh).
func TestSortLinesPriorityInKey(t *testing.T) {
	lines := []string{"1 (B) apple", "2 (A) zebra", "3 banana"}
	want := []string{"2 (A) zebra", "1 (B) apple", "3 banana"}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines = %v, want %v", lines, want)
	}
}

// Ties keep the original file order via the (zero-padded) line number.
func TestSortLinesTiesStable(t *testing.T) {
	lines := []string{"03 same text", "01 same text", "02 same text"}
	want := []string{"01 same text", "02 same text", "03 same text"}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines = %v, want %v", lines, want)
	}
}

// Zero-padded numbers (as produced by _format) keep multi-digit ties in file
// order; 10+ tasks exercise the padding.
func TestSortLinesTiesPadding(t *testing.T) {
	lines := []string{
		"01 aaa", "02 zzz", "03 aaa", "04 zzz", "05 aaa", "06 zzz",
		"07 aaa", "08 zzz", "09 aaa", "10 zzz", "11 aaa",
	}
	want := []string{
		"01 aaa", "03 aaa", "05 aaa", "07 aaa", "09 aaa", "11 aaa",
		"02 zzz", "04 zzz", "06 zzz", "08 zzz", "10 zzz",
	}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines = %v, want %v", lines, want)
	}
}

// Without uniform padding, the last-resort comparison is string-based, like
// sort's whole-line comparison: "10 aaa" sorts before "2 aaa". The pipeline
// avoids this by zero-padding (see TestSortLinesTiesPadding).
func TestSortLinesUnevenPadding(t *testing.T) {
	lines := []string{"2 aaa", "10 aaa", "1 aaa"}
	want := []string{"1 aaa", "10 aaa", "2 aaa"}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines = %v, want %v", lines, want)
	}
}

// The sort key starts at the field after the number and includes any leading
// blanks of the text — the behavior of the local (BSD) sort, verified
// against the real todo.sh.
func TestSortLinesLeadingBlanks(t *testing.T) {
	lines := []string{"1   foo", "2 !foo"}
	want := []string{"1   foo", "2 !foo"}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines = %v, want %v", lines, want)
	}
}

func TestSortLinesEmpty(t *testing.T) {
	var lines []string
	SortLines(lines)
	if lines != nil {
		t.Errorf("SortLines on empty input = %v, want nil", lines)
	}
}

// The pipeline zero-pads the numbered lines before sorting; the listing
// output shows the zero-padded numbers (t1300: "01 (A) ...", never " 1").
func TestZeroPadNumbers(t *testing.T) {
	lines := []string{" 1 line one", "11 line eleven", "  1 (A) @con01"}
	want := []string{"01 line one", "11 line eleven", "001 (A) @con01"}
	ZeroPadNumbers(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("ZeroPadNumbers = %v, want %v", lines, want)
	}
}

// Zero-padding makes space-padded numbered lines sortable: without it the
// key of " 1 line one" would start with the padding blank and sort by the
// number; zero-padded, all keys start with the text.
func TestSortLinesAfterZeroPad(t *testing.T) {
	lines := []string{" 9 zzz last", "10 aaa first"}
	ZeroPadNumbers(lines)
	want := []string{"10 aaa first", "09 zzz last"}
	SortLines(lines)
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("SortLines after ZeroPadNumbers = %v, want %v", lines, want)
	}
}
