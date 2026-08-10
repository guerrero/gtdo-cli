package todo

import (
	"fmt"
	"strings"
)

// SigilPolicy controls which parser-recognized context and project tokens may
// occur in a todo.txt line. A nil category is unrestricted; a non-nil slice,
// including an empty one, is an exact allow-list.
type SigilPolicy struct {
	AllowedContexts []string
	AllowedProjects []string
}

// SigilValidationError identifies the first disallowed sigil token in a line.
type SigilValidationError struct {
	Kind       string
	Token      string
	Path       string
	LineNumber int
}

// Error renders the todo.sh-compatible validation diagnostic.
func (e *SigilValidationError) Error() string {
	return fmt.Sprintf("TODO: %s %q is not allowed in %s at line %d.", e.Kind, e.Token, e.Path, e.LineNumber)
}

// Validate checks parser-recognized context and project words from left to
// right, returning the first token absent from its configured allow-list.
func (p SigilPolicy) Validate(path string, lineNumber int, text string) error {
	for _, word := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t'
	}) {
		if contextWordRe.MatchString(word) && !sigilAllowed(p.AllowedContexts, word) {
			return &SigilValidationError{Kind: "Context", Token: word, Path: path, LineNumber: lineNumber}
		}
		if projectWordRe.MatchString(word) && !sigilAllowed(p.AllowedProjects, word) {
			return &SigilValidationError{Kind: "Project", Token: word, Path: path, LineNumber: lineNumber}
		}
	}
	return nil
}

func sigilAllowed(allowed []string, token string) bool {
	if allowed == nil {
		return true
	}
	for _, candidate := range allowed {
		if candidate == token {
			return true
		}
	}
	return false
}
