package todo

import (
	"regexp"
	"strings"
)

// taskPrefix is the immutable metadata at the start of a todo.txt line.
// priority retains its trailing space because that is how the legacy parser
// exposes the prefix; uuid and date contain only their textual values.
type taskPrefix struct {
	done     bool
	priority string
	uuid     string
	date     string
	rest     string
}

var (
	// taskUUIDPrefixRe recognizes the canonical timestamp ID only in its
	// reserved prefix position. A final ID without a task body is valid too,
	// so the separator is either one space or the end of the line.
	taskUUIDPrefixRe = regexp.MustCompile(`^([0-9]{8}T[0-9]{6}\.[0-9]{2}Z)(?: |$)`)

	// legacyDatePrefixRe mirrors the loose year/month/day validation of the
	// original date parser while requiring a prefix boundary.
	legacyDatePrefixRe = regexp.MustCompile(`^((?:19|20)[0-9]{2}-[0-9]{2}-[0-9]{2})(?: |$)`)
)

// parseTaskPrefix splits the optional done marker, priority, timestamp ID,
// and legacy date from a task's body. Metadata is recognized in this exact
// order so an identifier in the body is never mistaken for a prefix.
func parseTaskPrefix(text string) taskPrefix {
	p := taskPrefix{rest: text}
	if strings.HasPrefix(text, donePrefix) {
		p.done = true
		text = text[len(donePrefix):]
	}
	if m := priorityRe.FindString(text); m != "" {
		p.priority = m
		text = text[len(m):]
	}
	if m := taskUUIDPrefixRe.FindStringSubmatch(text); m != nil {
		p.uuid = m[1]
		text = text[len(m[0]):]
	}
	if m := legacyDatePrefixRe.FindStringSubmatch(text); m != nil {
		p.date = m[1]
		text = text[len(m[0]):]
	}
	p.rest = text
	return p
}

// render rebuilds a line with this prefix and the supplied body. Metadata
// fields are emitted in canonical order, with a separator after each field.
func (p taskPrefix) render(rest string) string {
	var b strings.Builder
	if p.done {
		b.WriteString(donePrefix)
	}
	b.WriteString(p.priority)
	if p.uuid != "" {
		b.WriteString(p.uuid)
		b.WriteByte(' ')
	}
	if p.date != "" {
		b.WriteString(p.date)
		b.WriteByte(' ')
	}
	b.WriteString(rest)
	return b.String()
}
