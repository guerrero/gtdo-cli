package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	tokenRunes := line[start:pos]
	if len(tokenRunes) == 0 {
		return nil, 0
	}

	var candidates []string
	switch tokenRunes[0] {
	case '@':
		candidates = c.Candidates.Contexts
	case '+':
		candidates = c.Candidates.Projects
	default:
		return nil, 0
	}

	prefix := string(tokenRunes)
	matches := make([][]rune, 0, len(candidates))
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate, prefix) {
			continue
		}
		candidateRunes := []rune(candidate)
		suffix := append([]rune(nil), candidateRunes[len(tokenRunes):]...)
		matches = append(matches, suffix)
	}
	if len(matches) == 0 {
		return nil, 0
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return string(matches[i]) < string(matches[j])
	})
	return matches, len(tokenRunes)
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

// ttyAddInput is the readline-backed input boundary used when add receives a
// real terminal on stdin. The session writer is intentionally used for both
// readline output streams so prompts and redraws stay on stderr, matching the
// existing prompt contract.
type ttyAddInput struct {
	session  *session
	terminal *os.File
}

var _ addInput = ttyAddInput{}

// newAddInput selects the deterministic line protocol for pipes and tests and
// the readline adapter only for an actual terminal file. Keeping the session's
// buffered reader shared is important: a prior action read must not prefetch
// bytes that a later guided phase expects to consume.
func newAddInput(s *session, _ addCandidates) addInput {
	if s != nil && stdinIsTTY(s.in) {
		if terminal, ok := s.in.(*os.File); ok {
			return ttyAddInput{session: s, terminal: terminal}
		}
	}
	if s == nil {
		return lineAddInput{}
	}
	if s.reader == nil {
		s.reader = bufio.NewReader(s.in)
	}
	return lineAddInput{reader: s.reader}
}

// readlineConfig builds a fresh config for every prompt/selector. Readline's
// defaults target process stdin, while Cobra actions may receive another file,
// so raw-mode closures explicitly use the session terminal descriptor.
func (t ttyAddInput) readlineConfig(prompt string, completer readline.AutoCompleter, vimMode bool) *readline.Config {
	var state *readline.State
	fd := int(t.terminal.Fd())
	return &readline.Config{
		Prompt:                 prompt,
		HistoryFile:            "",
		HistoryLimit:           -1,
		DisableAutoSaveHistory: true,
		AutoComplete:           completer,
		Stdin:                  t.terminal,
		Stdout:                 t.session.errw,
		Stderr:                 t.session.errw,
		VimMode:                vimMode,
		FuncIsTerminal:         func() bool { return true },
		FuncMakeRaw: func() error {
			var err error
			state, err = readline.MakeRaw(fd)
			return err
		},
		FuncExitRaw: func() error {
			if state == nil {
				return nil
			}
			err := readline.Restore(fd, state)
			state = nil
			return err
		},
		FuncGetWidth:       func() int { return 80 },
		FuncOnWidthChanged: func(func()) {},
	}
}

// PromptTask reads the final task line with inline @context/+project
// completion. A readline instance is short-lived so add never persists
// history and terminal state is restored even when Readline returns EOF or
// Ctrl-C.
func (t ttyAddInput) PromptTask(candidates addCandidates) (string, error) {
	line, err := t.readline("Add: ", addCompleter{Candidates: candidates})
	return line, err
}

// PromptMetadata gathers zero or more key:value pairs. Empty keys finish the
// phase; an empty value skips that key and returns to the key prompt. Existing
// keys and values complete through readline, while custom entries remain
// supported after the completion list is exhausted.
func (t ttyAddInput) PromptMetadata(candidates addCandidates) ([]string, error) {
	keys := make([]string, 0, len(candidates.Metadata))
	valuesByKey := make(map[string][]string, len(candidates.Metadata))
	for _, candidate := range candidates.Metadata {
		keys = append(keys, candidate.Key)
		valuesByKey[candidate.Key] = append([]string(nil), candidate.Values...)
	}

	var metadata []string
	for {
		key, err := t.readline("Metadata key: ", stringCompleter{Candidates: keys})
		if err != nil {
			return nil, err
		}
		if key == "" {
			return metadata, nil
		}
		if !isMetadataKey(key) {
			return nil, fmt.Errorf("invalid metadata key %q", key)
		}

		value, err := t.readline("Metadata value: ", stringCompleter{Candidates: valuesByKey[key]})
		if err != nil {
			return nil, err
		}
		if value == "" {
			continue
		}
		if err := validateMetadataLine(key + ":" + value); err != nil {
			return nil, err
		}
		metadata = append(metadata, key+":"+value)
	}
}

// Select runs the raw searchable multi-select list for a project or context
// phase. The selector is terminal-independent at the state level, while this
// method owns rendering, key decoding, and raw-mode restoration.
func (t ttyAddInput) Select(_ guidedPhase, options []string) ([]string, error) {
	config := t.readlineConfig("", nil, true)
	terminal, err := readline.NewTerminal(config)
	if err != nil {
		return nil, err
	}
	defer func() { _ = terminal.Close() }()
	if err := terminal.EnterRawMode(); err != nil {
		return nil, err
	}
	defer func() { _ = terminal.ExitRawMode() }()

	state := newSelectorState(options)
	fmt.Fprintln(t.session.errw, selectorHelp)
	renderSelector(t.session.errw, state)
	terminal.KickRead()
	var pending *rune
	for {
		key := readSelectorKey(terminal, &pending)
		if key == 0 {
			return nil, io.EOF
		}
		action := state.handle(key)
		switch action {
		case selectorConfirm:
			return state.model.Values(), nil
		case selectorExit:
			return nil, nil
		case selectorCancel:
			return nil, &readline.InterruptError{}
		}
		renderSelector(t.session.errw, state)
		terminal.KickRead()
	}
}

func (t ttyAddInput) readline(prompt string, completer readline.AutoCompleter) (string, error) {
	instance, err := readline.NewEx(t.readlineConfig(prompt, completer, false))
	if err != nil {
		return "", err
	}
	defer func() { _ = instance.Close() }()
	return instance.Readline()
}

// stringCompleter offers suffixes for metadata key/value tokens. Unlike the
// task completer it accepts ordinary words and therefore works for custom
// metadata names without introducing sigil syntax.
type stringCompleter struct {
	Candidates []string
}

var _ readline.AutoCompleter = stringCompleter{}

func (c stringCompleter) Do(line []rune, pos int) ([][]rune, int) {
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
	prefixRunes := line[start:pos]
	prefix := string(prefixRunes)
	if prefix == "" {
		return nil, 0
	}
	var matches [][]rune
	for _, candidate := range c.Candidates {
		if !strings.HasPrefix(candidate, prefix) {
			continue
		}
		candidateRunes := []rune(candidate)
		matches = append(matches, append([]rune(nil), candidateRunes[len(prefixRunes):]...))
	}
	if len(matches) == 0 {
		return nil, 0
	}
	sort.SliceStable(matches, func(i, j int) bool { return string(matches[i]) < string(matches[j]) })
	return matches, len(prefixRunes)
}

func isMetadataKey(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if !isASCIIAlphaNumeric(char) {
			return false
		}
	}
	return true
}

const selectorHelp = "↑/↓ move  Space toggle  / filter  Enter confirm  Esc clear/exit"

type selectorAction uint8

const (
	selectorContinue selectorAction = iota
	selectorConfirm
	selectorExit
	selectorCancel
)

type selectorState struct {
	model     selectorModel
	queryMode bool
}

func newSelectorState(options []string) selectorState {
	return selectorState{model: newSelectorModel(options)}
}

func (s *selectorState) handle(key rune) selectorAction {
	switch key {
	case readline.CharPrev:
		s.model.Move(-1)
	case readline.CharNext:
		s.model.Move(1)
	case ' ':
		if s.queryMode {
			s.model.SetQuery(s.model.Query + " ")
		} else {
			s.model.Toggle()
		}
	case '/':
		s.queryMode = true
	case readline.CharBackspace, readline.CharCtrlH:
		if s.queryMode {
			query := []rune(s.model.Query)
			if len(query) > 0 {
				s.model.SetQuery(string(query[:len(query)-1]))
			}
		}
	case readline.CharEnter, readline.CharCtrlJ:
		return selectorConfirm
	case readline.CharEsc:
		if s.model.Query != "" {
			s.model.SetQuery("")
			s.queryMode = false
			return selectorContinue
		}
		s.queryMode = false
		return selectorExit
	case readline.CharInterrupt:
		return selectorCancel
	default:
		if s.queryMode && readline.IsPrintable(key) {
			s.model.SetQuery(s.model.Query + string(key))
		}
	}
	return selectorContinue
}

func renderSelector(w io.Writer, state selectorState) {
	// Clear the previous selector frame. The output is intentionally kept on
	// stderr with readline's normal prompt stream; non-TTY adapters never call
	// this path and therefore emit no escape sequences.
	fmt.Fprint(w, "\r\033[2K")
	visible := state.model.Visible()
	for i, option := range visible {
		marker := "[ ]"
		if state.model.Selected[option] {
			marker = "[x]"
		}
		cursor := "  "
		if i == state.model.Cursor {
			cursor = "> "
		}
		fmt.Fprintf(w, "\033[2K%s%s %s\n", cursor, marker, option)
	}
	if state.queryMode {
		fmt.Fprintf(w, "/%s", state.model.Query)
	}
}

// readSelectorKey decodes arrow escape sequences while keeping a plain Esc
// available to selectorState. readline's Terminal normally reserves Esc as
// the prefix of an escape sequence; VimMode is enabled for this raw terminal
// so we can distinguish a lone Esc and decode the small set of arrows needed
// by the selector ourselves.
func readSelectorKey(terminal *readline.Terminal, pending **rune) rune {
	if *pending != nil {
		key := **pending
		*pending = nil
		return key
	}
	key := terminal.ReadRune()
	if key != readline.CharEsc {
		return key
	}
	next := terminal.ReadRune()
	if next == 0 {
		return key
	}
	if next != '[' && next != 'O' {
		*pending = &next
		return key
	}
	final := terminal.ReadRune()
	switch final {
	case 'A':
		return readline.CharPrev
	case 'B':
		return readline.CharNext
	default:
		return final
	}
}
