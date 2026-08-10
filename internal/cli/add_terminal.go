package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

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
	state    *ttyInputState
}

var _ addInput = (*ttyAddInput)(nil)

type ttyInputState struct {
	promptConfig *readline.Config
	prompt       *readline.Instance
	promptSource *os.File
	promptInput  *readline.CancelableStdin
	promptRaw    *readline.State
	selectorKeys chan rune
	selectorStop chan struct{}
	selectorPrev *readline.Config
	selectorPend *rune
	close        sync.Once
}

// newAddInput selects the deterministic line protocol for pipes and tests and
// the readline adapter only for an actual terminal file. Keeping the session's
// buffered reader shared is important: a prior action read must not prefetch
// bytes that a later guided phase expects to consume.
func newAddInput(s *session, _ addCandidates) addInput {
	if s != nil && stdinIsTTY(s.in) {
		if terminal, ok := s.in.(*os.File); ok {
			return &ttyAddInput{session: s, terminal: terminal}
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

// ensurePrompt creates one readline instance for the complete prompt portion
// of an add session. Its terminal owns a buffered input reader; keeping that
// boundary alive across prompts preserves typeahead (including pasted guided
// phase lines) instead of discarding bytes when a per-prompt instance closes.
func (t *ttyAddInput) ensureTerminal() (*ttyInputState, error) {
	if t.state != nil {
		return t.state, nil
	}
	state := &ttyInputState{}
	source, err := duplicateTTY(t.terminal)
	if err != nil {
		return nil, err
	}
	state.promptSource = source
	state.promptInput = readline.NewCancelableStdin(source)
	state.promptConfig = t.newConfig(state.promptInput, int(source.Fd()), func() *readline.State {
		return state.promptRaw
	}, func(raw *readline.State) {
		state.promptRaw = raw
	})
	state.prompt, err = readline.NewEx(state.promptConfig)
	if err != nil {
		_ = state.promptInput.Close()
		_ = source.Close()
		return nil, err
	}
	t.state = state
	return state, nil
}

// newConfig supplies explicit terminal callbacks so tests can run readline
// against a duplicated PTY descriptor without touching Cobra's stdin.
func (t *ttyAddInput) newConfig(source io.ReadCloser, fd int, raw func() *readline.State, setRaw func(*readline.State)) *readline.Config {
	config := &readline.Config{
		Prompt:                 "",
		HistoryFile:            "",
		HistoryLimit:           -1,
		DisableAutoSaveHistory: true,
		Stdin:                  source,
		Stdout:                 t.session.errw,
		Stderr:                 t.session.errw,
		Painter:                ttyPainter{},
		VimMode:                false,
		FuncIsTerminal:         func() bool { return true },
		FuncMakeRaw: func() error {
			if raw() != nil {
				return nil
			}
			rawState, err := readline.MakeRaw(fd)
			if err == nil {
				setRaw(rawState)
			}
			return err
		},
		FuncExitRaw: func() error {
			rawState := raw()
			if rawState == nil {
				return nil
			}
			err := readline.Restore(fd, rawState)
			setRaw(nil)
			return err
		},
		FuncGetWidth:       func() int { return 80 },
		FuncOnWidthChanged: func(func()) {},
	}
	return config
}

type ttyPainter struct{}

func (ttyPainter) Paint(line []rune, _ int) []rune { return line }

func duplicateTTY(file *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), file.Name()), nil
}

// Close restores raw mode and stops readline's input goroutine. A
// CancelableStdin is closed before its duplicated descriptor so a blocked PTY
// read is interrupted without closing Cobra's original stdin file.
func (t *ttyAddInput) Close() error {
	if t == nil || t.state == nil {
		return nil
	}
	var err error
	t.state.close.Do(func() {
		if t.state.promptRaw != nil && t.state.prompt != nil {
			if rawErr := t.state.prompt.Terminal.ExitRawMode(); err == nil {
				err = rawErr
			}
		}
		if t.state.selectorStop != nil {
			if t.state.prompt != nil && t.state.selectorPrev != nil {
				t.state.prompt.SetConfig(t.state.selectorPrev)
			}
			close(t.state.selectorStop)
			t.state.selectorStop = nil
			t.state.selectorKeys = nil
		}
		if t.state.promptSource != nil {
			if t.state.promptInput != nil {
				_ = t.state.promptInput.Close()
			}
			_ = t.state.promptSource.Close()
		}
		if t.state.prompt != nil {
			if closeErr := t.state.prompt.Close(); err == nil {
				err = closeErr
			}
		}
	})
	return err
}

// PromptTask reads the final task line with inline @context/+project
// completion. The shared readline instance keeps typeahead available for
// subsequent guided phases while history remains disabled.
func (t *ttyAddInput) PromptTask(candidates addCandidates) (string, error) {
	line, err := t.readline("Add: ", addCompleter{Candidates: candidates})
	return line, err
}

// PromptMetadata gathers zero or more key:value pairs. Empty keys finish the
// phase; an empty value skips that key and returns to the key prompt. Existing
// keys and values complete through readline, while custom entries remain
// supported after the completion list is exhausted.
func (t *ttyAddInput) PromptMetadata(candidates addCandidates) ([]string, error) {
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
func (t *ttyAddInput) Select(_ guidedPhase, options []string) ([]string, error) {
	state, err := t.ensureTerminal()
	if err != nil {
		return nil, err
	}
	if state.prompt == nil {
		return nil, io.EOF
	}
	if state.selectorKeys == nil {
		state.selectorKeys = make(chan rune, 64)
		state.selectorStop = make(chan struct{})
		state.selectorPrev = state.prompt.Config.Clone()
		selectorConfig := state.prompt.Config.Clone()
		selectorConfig.Prompt = ""
		selectorConfig.AutoComplete = nil
		selectorConfig.VimMode = true
		keys := state.selectorKeys
		stop := state.selectorStop
		selectorConfig.FuncFilterInputRune = func(key rune) (rune, bool) {
			select {
			case keys <- key:
				// Let readline's operation observe EOF itself; filtering rune(0)
				// would otherwise make its goroutine spin on a closed terminal.
				return 0, key == 0
			case <-stop:
				return 0, false
			}
		}
		state.prompt.SetConfig(selectorConfig)
	}
	terminal := state.prompt.Terminal
	if err := terminal.EnterRawMode(); err != nil {
		return nil, err
	}
	defer func() {
		_ = terminal.ExitRawMode()
	}()

	selector := newSelectorState(options)
	fmt.Fprintln(t.session.errw, selectorHelp)
	renderSelector(t.session.errw, &selector)
	terminal.KickRead()
	reader := selectorChannelReader{keys: state.selectorKeys, stop: state.selectorStop}
	for {
		key := readSelectorKey(reader, &state.selectorPend)
		if key == 0 {
			return nil, io.EOF
		}
		action := selector.handle(key)
		switch action {
		case selectorConfirm:
			return selector.model.Values(), nil
		case selectorExit:
			return nil, nil
		case selectorCancel:
			return nil, &readline.InterruptError{}
		}
		renderSelector(t.session.errw, &selector)
		terminal.KickRead()
	}
}

// selectorChannelReader receives runes from readline's one background
// operation. It keeps the terminal's buffered input boundary intact while
// allowing the selector state machine to consume keys independently.
type selectorChannelReader struct {
	keys <-chan rune
	stop <-chan struct{}
}

func (r selectorChannelReader) ReadSelectorKey() rune {
	select {
	case key := <-r.keys:
		return key
	case <-r.stop:
		return 0
	}
}

func (r selectorChannelReader) ReadRuneTimeout(timeout time.Duration) (rune, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case key := <-r.keys:
		return key, key != 0
	case <-r.stop:
		return 0, false
	case <-timer.C:
		return 0, false
	}
}

func (t *ttyAddInput) readline(prompt string, completer readline.AutoCompleter) (string, error) {
	state, err := t.ensureTerminal()
	if err != nil {
		return "", err
	}
	if state.prompt == nil {
		return "", io.EOF
	}
	state.prompt.SetPrompt(prompt)
	state.prompt.SetVimMode(false)
	config := state.prompt.Config.Clone()
	config.Prompt = prompt
	config.AutoComplete = completer
	config.VimMode = false
	state.prompt.SetConfig(config)
	return state.prompt.Readline()
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
	model         selectorModel
	queryMode     bool
	renderedLines int
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
	case readline.CharInterrupt, readline.CharDelete:
		return selectorCancel
	default:
		if s.queryMode && readline.IsPrintable(key) {
			s.model.SetQuery(s.model.Query + string(key))
		}
	}
	return selectorContinue
}

func renderSelector(w io.Writer, state *selectorState) {
	visible := state.model.Visible()
	lines := make([]string, 0, len(visible)+1)
	for i, option := range visible {
		marker := "[ ]"
		if state.model.Selected[option] {
			marker = "[x]"
		}
		cursor := "  "
		if i == state.model.Cursor {
			cursor = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s %s", cursor, marker, option))
	}
	if state.queryMode {
		lines = append(lines, "/"+state.model.Query)
	}
	if state.renderedLines > 0 {
		fmt.Fprintf(w, "\033[%dA", state.renderedLines)
	}
	lineCount := state.renderedLines
	if len(lines) > lineCount {
		lineCount = len(lines)
	}
	for i := 0; i < lineCount; i++ {
		fmt.Fprint(w, "\r\033[2K")
		if i < len(lines) {
			fmt.Fprint(w, lines[i])
		}
		fmt.Fprint(w, "\n")
	}
	state.renderedLines = lineCount
}

// readSelectorKey decodes arrow escape sequences while keeping a plain Esc
// available to selectorState. readline's Terminal normally reserves Esc as
// the prefix of an escape sequence; VimMode is enabled for this raw terminal
// so we can distinguish a lone Esc and decode the small set of arrows needed
// by the selector ourselves.
type selectorRuneReader interface {
	ReadSelectorKey() rune
}

type selectorTimedRuneReader interface {
	ReadRuneTimeout(time.Duration) (rune, bool)
}

const selectorEscapeWait = 20 * time.Millisecond

func readSelectorKey(terminal selectorRuneReader, pending **rune) rune {
	if pending != nil && *pending != nil {
		key := **pending
		*pending = nil
		return key
	}
	key := terminal.ReadSelectorKey()
	if key != readline.CharEsc {
		return key
	}
	next, ok := readSelectorRune(terminal, selectorEscapeWait)
	if !ok {
		return key
	}
	if next != '[' && next != 'O' {
		if pending != nil {
			*pending = &next
		}
		return key
	}
	final, ok := readSelectorRune(terminal, selectorEscapeWait)
	if !ok {
		return key
	}
	switch final {
	case 'A':
		return readline.CharPrev
	case 'B':
		return readline.CharNext
	default:
		return final
	}
}

func readSelectorRune(reader selectorRuneReader, timeout time.Duration) (rune, bool) {
	if timed, ok := reader.(selectorTimedRuneReader); ok {
		return timed.ReadRuneTimeout(timeout)
	}
	key := reader.ReadSelectorKey()
	return key, key != 0
}
