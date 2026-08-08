package todo

// Sorting of numbered task lines per plan §6.2.3: case-insensitive by the
// task text, ties broken by the original file order via the zero-padded
// line numbers, replicating `LC_COLLATE=C sort -f -k2` (see §10).

import (
	"sort"
	"strings"
)

// SortLines sorts numbered list lines the way `LC_COLLATE=C sort -f -k2`
// does: by the task text after the line number, case-insensitively. Ties
// fall back to the full line, exactly like sort's last-resort comparison,
// so the zero-padded line numbers the _format pipeline produces keep the
// original file order.
func SortLines(lines []string) {
	pairs := make([]lineKey, len(lines))
	for i, line := range lines {
		pairs[i] = lineKey{line: line, key: foldASCII(sortKey(line))}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key || (pairs[i].key == pairs[j].key && pairs[i].line < pairs[j].line)
	})
	for i, p := range pairs {
		lines[i] = p.line
	}
}

// lineKey pairs a numbered line with its precomputed sort key so that the
// key stays attached to its line while SliceStable permutes the slice.
type lineKey struct {
	line string
	key  string
}

// ZeroPadNumbers replaces the leading spaces of each numbered line with
// zeros, the _format step that lets `sort -f -k2` break ties by line number
// (todo.sh's `s/^ /0/` chain). Lines numbered with NumberedLine carry the
// number space-padded; zero-padding to the numbering width makes ties sort
// in original file order and is what the listing output shows ("01", not
// " 1"). todo.sh only pads up to five leading spaces; this pads any width.
func ZeroPadNumbers(lines []string) {
	for i, line := range lines {
		n := 0
		for n < len(line) && line[n] == ' ' {
			n++
		}
		lines[i] = strings.Repeat("0", n) + line[n:]
	}
}

// sortKey returns the -k2 field of a numbered line: everything from the
// first blank after the number on, including any leading blanks of the
// text. That matches the local sort's field handling, verified against the
// real todo.sh. A line without blanks has no second field.
func sortKey(line string) string {
	i := strings.IndexAny(line, " \t")
	if i < 0 {
		return ""
	}
	return line[i:]
}

// foldASCII lowercases ASCII letters only, like `sort -f` under
// LC_COLLATE=C; Unicode case folding would diverge from todo.sh for
// non-ASCII text.
func foldASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
