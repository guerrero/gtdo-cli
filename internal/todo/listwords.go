package todo

// The listing extras beyond the _format pipeline: the listcon/listproj
// word extraction (todo.sh's listWordsWithSigil) and the listpri priority
// class. Both mirror grep's exact behavior on the default todo.cfg
// settings; the SIGIL_* customization patterns are out of scope (plan §2).

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SigilWords returns the unique sigil words of the lines that pass the
// term filters, exactly like todo.sh's listWordsWithSigil with the default
// sigil patterns (before/after empty, valid `.*`): the terms filter the
// raw lines (grep -i / grep -v -i, todo.sh:1058), grep -o "[^ ]*@[^ ]\+"
// extracts one match per maximal non-space run, and sort -u dedupes
// byte-wise. A run matches grep -o iff it contains the sigil somewhere
// before its last character; the default sed then keeps only the matches
// that start with the sigil — so the net effect is: every run starting
// with the sigil and having at least one more character, whole, even when
// it holds further sigils ("@con05@con06" is one context). Runs that
// merely contain the sigil ("w:@Other", "(@school") are not listed; the
// split is on spaces only, since [^ ] spans tabs.
func SigilWords(lines []string, sigil byte, terms []string) ([]string, error) {
	filtered, err := FilterLines(lines, terms)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var out []string
	for _, line := range filtered {
		for _, run := range strings.Split(line, " ") {
			if len(run) >= 2 && run[0] == sigil && !seen[run] {
				seen[run] = true
				out = append(out, run)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// PriorityClass is a compiled listpri priority filter: the character class
// of todo.sh's `grep '^ *[0-9]\+ ([${pri}]) '` post-filter (todo.sh:1384),
// which embeds the PRIORITIES argument inside a bracket expression.
type PriorityClass func(letter byte) bool

// CompilePriorityClass compiles a listpri PRIORITIES argument the way GNU
// grep compiles the `[${pri}]` bracket expression (BRE, C locale): letters
// are literals, X-Y between two letters is an ascending range, and a
// reversed range (Z-A), a dash after a completed range (A-B-C), or a dash
// before another dash (A--Z) is the "invalid character range" error that
// fails the whole post-filter grep (todo.sh then shows 0 tasks).
func CompilePriorityClass(pri string) (PriorityClass, error) {
	var out []byte
	prevRange := false
	for i := 0; i < len(pri); i++ {
		c := pri[i]
		if c != '-' {
			out = append(out, c)
			prevRange = false
			continue
		}
		if i == 0 || i+1 >= len(pri) {
			// A bracket expression may lead or trail with a literal dash,
			// but no matched PRIORITIES argument can (the listpri grep
			// always starts and ends with a letter).
			return nil, errInvalidRange
		}
		next := pri[i+1]
		if prevRange {
			// [A-B-C]: the dash follows a completed range.
			return nil, errInvalidRange
		}
		if next == '-' {
			// [A--Z]: the range A..- is reversed.
			return nil, errInvalidRange
		}
		prev := pri[i-1]
		if prev > next {
			// [Z-A]: reversed range.
			return nil, errInvalidRange
		}
		for l := prev; l <= next; l++ {
			out = append(out, l)
		}
		prevRange = true
		i++ // the range consumed the next letter
	}
	return func(letter byte) bool {
		for _, c := range out {
			if c == letter {
				return true
			}
		}
		return false
	}, nil
}

// errInvalidRange is GNU grep's diagnostic when a bracket expression fails
// to compile; listpri's post-filter grep prints it and matches nothing.
var errInvalidRange = fmt.Errorf("grep: invalid character range")

// listpriArgRe is the grep of todo.sh's listpri priority detection,
// `^\([A-Za-z]\|[A-Za-z]-[A-Za-z]\|[A-Z][A-Z-]*[A-Z]\)$` translated to
// RE2: a single letter, a letter-dash-letter, or an uppercase sequence of
// letters and dashes starting and ending with a letter.
var listpriArgRe = regexp.MustCompile(`^([A-Za-z]|[A-Za-z]-[A-Za-z]|[A-Z][A-Z-]*[A-Z])$`)

// ListpriArg reports whether arg is a PRIORITIES argument and returns it
// uppercased (tr '[:lower:]' '[:upper:]'). An empty or non-matching arg is
// not a PRIORITIES argument and stays a filter term (todo.sh:1382-1383).
func ListpriArg(arg string) (string, bool) {
	if !listpriArgRe.MatchString(arg) {
		return "", false
	}
	return strings.ToUpper(arg), true
}
