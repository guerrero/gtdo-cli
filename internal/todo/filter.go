package todo

// Term filtering of numbered task lines (plan §6.2.2): terms are AND'ed with
// case-insensitive grep semantics, a leading "-" turns a term into an
// exclusion (grep -v), and "\|" inside a term is an OR of alternates. The
// terms are basic regexes (BRE); translateBRE converts them to RE2, since
// RE2 has no BRE mode.

import (
	"regexp"
	"strings"
)

// CompileTerm translates a filter term from grep basic regex syntax to RE2
// and compiles it case-insensitively (grep -i). An empty term matches every
// line, like `grep ""`.
func CompileTerm(term string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i)" + translateBRE(term))
}

// MatchLine reports whether a numbered line matches one filter term:
// case-insensitive basic-regex matching anywhere in the line, exactly what
// the `grep -i` steps of todo.sh's filtercommand do to the numbered lines.
func MatchLine(line, term string) (bool, error) {
	re, err := CompileTerm(term)
	if err != nil {
		return false, err
	}
	return re.MatchString(line), nil
}

// FilterLines applies todo.sh's filtercommand to numbered lines: the terms
// are AND'ed in order, a leading "-" on a term turns it into an exclusion
// (the first dash is stripped, like `${search_term:1}`), and "\|" inside a
// term is an OR of alternates. Matching is case-insensitive and the line
// number participates, because the pipeline filters the numbered lines.
// Line order is preserved; a bare "-" term excludes everything, replicating
// `grep -v -i ""`.
func FilterLines(lines, terms []string) ([]string, error) {
	out := lines
	for _, term := range terms {
		exclude := strings.HasPrefix(term, "-")
		if exclude {
			term = term[1:]
		}
		re, err := CompileTerm(term)
		if err != nil {
			return nil, err
		}
		kept := make([]string, 0, len(out))
		for _, line := range out {
			if re.MatchString(line) != exclude {
				kept = append(kept, line)
			}
		}
		out = kept
	}
	return out, nil
}

// translateBRE converts a grep basic regular expression (BRE) to the
// equivalent RE2 pattern. BRE and RE2 disagree on what is special: BRE
// needs backslashes for the operators ( ) + ? { | and } (GNU extensions),
// while RE2 treats the bare characters as operators — so bare characters
// are escaped and escaped operators unescaped. Escapes both engines share
// (\. \* \\ \b \w \s \d \t \n and friends, matching the local grep) pass
// through unchanged; a backslash before any other character is dropped,
// because grep treats it as that literal character (verified with the
// system grep). grep's \1..\9 backreferences have no RE2 equivalent and
// approximate as the literal digit (plan §10).
func translateBRE(term string) string {
	var b strings.Builder
	for i := 0; i < len(term); i++ {
		c := term[i]
		if c == '\\' {
			if i+1 >= len(term) {
				b.WriteByte(c) // trailing backslash: both engines reject it
				continue
			}
			n := term[i+1]
			switch n {
			case '(', ')', '|', '+', '?', '{', '}':
				b.WriteByte(n) // BRE operator → RE2 operator
			case 'b', 'B', '<', '>', 'w', 'W', 's', 'S', 'd', 'D',
				't', 'n', 'r', 'f', 'v', 'a', 'x',
				'.', '*', '\\', '[', ']', '^', '$':
				b.WriteByte(c)
				b.WriteByte(n) // shared escape, same meaning in RE2
			default:
				b.WriteByte(n) // grep: backslash before an ordinary char
			}
			i++
			continue
		}
		switch c {
		case '(', ')', '+', '?', '{', '}', '|':
			b.WriteByte('\\')
			b.WriteByte(c) // literal in BRE, operator in RE2
		case '$':
			// $ is an anchor only at the end of the term or before a closing
			// \) or \|; elsewhere grep treats it as a literal. RE2 anchors
			// mid-pattern too, so a literal $ must be escaped ("$$" would
			// otherwise match every line, not just lines ending in $).
			if i+1 == len(term) ||
				(term[i+1] == '\\' && i+2 < len(term) && (term[i+2] == ')' || term[i+2] == '|')) {
				b.WriteByte(c)
			} else {
				b.WriteByte('\\')
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
