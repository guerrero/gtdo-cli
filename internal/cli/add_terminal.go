package cli

import (
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

// addCompleter supplies context and project suffixes for the token at the
// readline cursor. The candidate lists are collected and sorted elsewhere.
type addCompleter struct {
	Candidates addCandidates
}

var _ readline.AutoCompleter = addCompleter{}

// Do implements readline.AutoCompleter for @context and +project tokens.
// Readline replaces the consumed token prefix with one of the returned
// suffixes, so words outside those two categories are left untouched.
func (c addCompleter) Do(line []rune, pos int) ([][]rune, int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(line) {
		pos = len(line)
	}

	start := pos
	for start > 0 && line[start-1] != ' ' && line[start-1] != '\t' {
		start--
	}
	token := line[start:pos]
	if len(token) == 0 {
		return nil, 0
	}

	var candidates []string
	switch token[0] {
	case '@':
		candidates = c.Candidates.Contexts
	case '+':
		candidates = c.Candidates.Projects
	default:
		return nil, 0
	}

	prefix := string(token)
	matches := make([][]rune, 0, len(candidates))
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate, prefix) {
			continue
		}
		candidateRunes := []rune(candidate)
		suffix := append([]rune(nil), candidateRunes[len(token):]...)
		matches = append(matches, suffix)
	}
	if len(matches) == 0 {
		return nil, 0
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return string(matches[i]) < string(matches[j])
	})
	return matches, len(token)
}

// selectorModel is the terminal-independent state for a filtered
// multi-select list. Options are expected to arrive in sorted order.
type selectorModel struct {
	Options  []string
	Selected map[string]bool
	Cursor   int
	Query    string
}

func newSelectorModel(options []string) selectorModel {
	return selectorModel{
		Options:  append([]string(nil), options...),
		Selected: make(map[string]bool),
	}
}

// Move shifts the cursor by delta and clamps it to the currently visible
// options. An empty filter result keeps the cursor at zero.
func (m *selectorModel) Move(delta int) {
	visible := m.Visible()
	if len(visible) == 0 {
		m.Cursor = 0
		return
	}

	m.Cursor += delta
	m.clampCursor(len(visible))
}

// Toggle flips the selected state of the currently visible exact token.
func (m *selectorModel) Toggle() {
	visible := m.Visible()
	if len(visible) == 0 {
		m.Cursor = 0
		return
	}
	m.clampCursor(len(visible))

	if m.Selected == nil {
		m.Selected = make(map[string]bool)
	}
	token := visible[m.Cursor]
	m.Selected[token] = !m.Selected[token]
}

// SetQuery replaces the case-sensitive substring filter and keeps the cursor
// valid for the resulting visible list.
func (m *selectorModel) SetQuery(query string) {
	m.Query = query
	m.clampCursor(len(m.Visible()))
}

// Visible returns options containing Query as a case-sensitive substring.
func (m selectorModel) Visible() []string {
	if m.Query == "" {
		return append([]string(nil), m.Options...)
	}

	visible := make([]string, 0, len(m.Options))
	for _, option := range m.Options {
		if strings.Contains(option, m.Query) {
			visible = append(visible, option)
		}
	}
	if len(visible) == 0 {
		return nil
	}
	return visible
}

// Values returns selected options in their original sorted option order.
func (m selectorModel) Values() []string {
	if len(m.Selected) == 0 {
		return nil
	}

	values := make([]string, 0, len(m.Selected))
	for _, option := range m.Options {
		if m.Selected[option] {
			values = append(values, option)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (m *selectorModel) clampCursor(visible int) {
	if visible == 0 {
		m.Cursor = 0
		return
	}
	if m.Cursor < 0 {
		m.Cursor = 0
		return
	}
	if m.Cursor >= visible {
		m.Cursor = visible - 1
	}
}
