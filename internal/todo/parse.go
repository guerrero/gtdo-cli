package todo

// Parsing of todo.txt lines into Tasks, per todo.sh conventions: the
// priority regex `^\([A-Z]\) `, the `(19|20)xx-xx-xx` date regex, the `x `
// done marker, and the @context/+project sigil words (plan §6.2.4, §6.3).
// The shell BREs translate directly to RE2 and are pinned by the tests
// against the todo.sh test suite.

import (
	"regexp"
	"strings"
)

var (
	// priorityRe matches a todo.sh priority at the start of a task: an
	// uppercase letter in parens followed by a space. Lowercase letters are
	// not priorities, matching listpri and the _format awk.
	priorityRe = regexp.MustCompile(`^\(([A-Z])\) `)

	// dateRe matches todo.sh's date at the start of a task. The plan pins
	// `(19|20)\d\d-\d\d-\d\d`: the year must start with 19 or 20, and month
	// and day ranges are not validated (todo.sh's priAndDateExpr is equally
	// loose).
	dateRe = regexp.MustCompile(`^(19|20)\d\d-\d\d-\d\d`)

	// donePrefix marks a completed task; todo.sh tests for the two literal
	// characters "x " (case-sensitive) in `do` and archive.
	donePrefix = "x "

	// contextWordRe and projectWordRe classify whitespace-delimited words as
	// sigil words: they must start with the sigil and end in [A-Za-z0-9_],
	// the same rule the _format awk uses to color contexts and projects
	// (§6.2.4). "@con05@con06" is one word and therefore one context.
	contextWordRe = regexp.MustCompile(`^@.*[A-Za-z0-9_]$`)
	projectWordRe = regexp.MustCompile(`^\+.*[A-Za-z0-9_]$`)
)

// Done reports whether the task carries the "x " completion marker.
func (t Task) Done() bool {
	return strings.HasPrefix(t.Text, donePrefix)
}

// Priority returns the task's priority letter A-Z and whether the task has
// one. The priority must be at the very start of the line: "(A) " in the
// middle of the text, or after the "x " marker, is not a priority, matching
// listpri and the _format awk.
func (t Task) Priority() (byte, bool) {
	m := priorityRe.FindStringSubmatch(t.Text)
	if m == nil {
		return 0, false
	}
	return m[1][0], true
}

// Date returns the legacy YYYY-MM-DD date at the start of the task: the
// creation date after optional priority/ID metadata, or the completion date
// after the "x " marker. Canonical IDs are skipped rather than returned as
// dates.
func (t Task) Date() string {
	p := parseTaskPrefix(t.Text)
	if p.date != "" {
		return p.date
	}
	// Keep the historical unbounded dateRe behavior for legacy lines whose
	// date is followed by a non-space delimiter. The shared prefix parser is
	// intentionally stricter so render can preserve metadata boundaries.
	return dateRe.FindString(p.rest)
}

// Contexts returns the @-sigil words found in the text, sigil included, in
// order of appearance. Duplicates are kept: deduplication is listcon's job
// (Task 8).
func (t Task) Contexts() []string {
	return sigilWords(t.Text, contextWordRe)
}

// Projects returns the +-sigil words found in the text, sigil included, in
// order of appearance. Duplicates are kept: deduplication is listproj's job
// (Task 8).
func (t Task) Projects() []string {
	return sigilWords(t.Text, projectWordRe)
}

// sigilWords splits text on runs of spaces and tabs — the same word
// boundaries the _format awk uses — and keeps the words classified by re.
func sigilWords(text string, re *regexp.Regexp) []string {
	var words []string
	for _, w := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t'
	}) {
		if re.MatchString(w) {
			words = append(words, w)
		}
	}
	return words
}
