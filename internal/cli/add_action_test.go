package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
)

func TestActionAddGuidedUsesStoreOnce(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Dir:        dir,
		TodoFile:   filepath.Join(dir, "todo.txt"),
		DoneFile:   filepath.Join(dir, "done.txt"),
		ReportFile: filepath.Join(dir, "report.txt"),
		Verbose:    1,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("Call team\n\n\n\n"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := actionAdd(cmd, []string{"--guided"}, cfg); err != nil {
		t.Fatalf("actionAdd: %v", err)
	}
	got, err := os.ReadFile(cfg.TodoFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Call team\n" {
		t.Fatalf("todo.txt = %q, want one final add", got)
	}
	if got := stdout.String(); got != "1 Call team\nTODO: 1 added.\n" {
		t.Fatalf("stdout = %q, want one confirmation", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for line input", stderr.String())
	}
}

func TestActionAddModeErrorsDoNotEnsureFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Dir:        dir,
		TodoFile:   filepath.Join(dir, "todo.txt"),
		DoneFile:   filepath.Join(dir, "done.txt"),
		ReportFile: filepath.Join(dir, "report.txt"),
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := actionAdd(cmd, []string{"--guided", "--only", "unknown"}, cfg)
	if err == nil {
		t.Fatal("actionAdd succeeded for an invalid mode")
	}
	if _, statErr := os.Stat(cfg.TodoFile); !os.IsNotExist(statErr) {
		t.Fatalf("todo file stat error = %v, want absent file", statErr)
	}
	if got, want := stderr.String(), addUsage()+"\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestActionAddForceRejectsExplicitModeWithoutEnsuringFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Dir:        dir,
		TodoFile:   filepath.Join(dir, "todo.txt"),
		DoneFile:   filepath.Join(dir, "done.txt"),
		ReportFile: filepath.Join(dir, "report.txt"),
		Force:      true,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("task\n"))
	var stderr bytes.Buffer
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(&stderr)

	if err := actionAdd(cmd, []string{"-i"}, cfg); err == nil {
		t.Fatal("actionAdd succeeded for -f with an explicit mode")
	}
	if _, err := os.Stat(cfg.TodoFile); !os.IsNotExist(err) {
		t.Fatalf("todo file stat error = %v, want absent file", err)
	}
	if got, want := stderr.String(), addUsage()+"\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestActionAddGuidedEOFDoesNotMutateTodo(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Dir:        dir,
		TodoFile:   filepath.Join(dir, "todo.txt"),
		DoneFile:   filepath.Join(dir, "done.txt"),
		ReportFile: filepath.Join(dir, "report.txt"),
	}
	if err := os.WriteFile(cfg.TodoFile, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("partial"))
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := actionAdd(cmd, []string{"--guided"}, cfg); err == nil {
		t.Fatal("actionAdd succeeded after EOF")
	}
	got, err := os.ReadFile(cfg.TodoFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing\n" {
		t.Fatalf("todo.txt = %q, want unchanged task file", got)
	}
}

func TestNewAddInputUsesLineAdapterForNonTTY(t *testing.T) {
	s := &session{in: strings.NewReader("task\n"), reader: nil}
	input := newAddInput(s, addCandidates{})
	if _, ok := input.(lineAddInput); !ok {
		t.Fatalf("newAddInput() = %T, want lineAddInput", input)
	}
}

func TestSelectorStateEscClearsQueryBeforeExit(t *testing.T) {
	state := newSelectorState([]string{"@home", "@office"})
	state.model.SetQuery("of")
	state.queryMode = true
	if got := state.handle(readline.CharEsc); got != selectorContinue {
		t.Fatalf("first Esc action = %v, want continue", got)
	}
	if state.model.Query != "" || state.queryMode {
		t.Fatalf("first Esc state = query %q, queryMode %t; want cleared filter", state.model.Query, state.queryMode)
	}
	if got := state.handle(readline.CharEsc); got != selectorExit {
		t.Fatalf("second Esc action = %v, want exit", got)
	}
}

func TestSelectorStateEnterConfirmsSelection(t *testing.T) {
	state := newSelectorState([]string{"@home"})
	state.model.Toggle()
	if got := state.handle(readline.CharEnter); got != selectorConfirm {
		t.Fatalf("Enter action = %v, want confirm", got)
	}
	if got := state.model.Values(); len(got) != 1 || got[0] != "@home" {
		t.Fatalf("selected values = %v, want [@home]", got)
	}
}

func TestSelectorStateCtrlDIsCancellation(t *testing.T) {
	state := newSelectorState([]string{"@home"})
	if got := state.handle(readline.CharDelete); got != selectorCancel {
		t.Fatalf("Ctrl-D action = %v, want cancellation", got)
	}
}

func TestReadSelectorKeyDoesNotWaitForLoneEsc(t *testing.T) {
	reader := &selectorRuneReaderStub{keys: []rune{readline.CharEsc}}
	done := make(chan rune, 1)
	go func() { done <- readSelectorKey(reader, nil) }()
	select {
	case got := <-done:
		if got != readline.CharEsc {
			t.Fatalf("readSelectorKey() = %d, want Esc", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("readSelectorKey waited for a second rune after Esc")
	}
}

func TestRenderSelectorClearsPreviousFrame(t *testing.T) {
	state := newSelectorState([]string{"@home", "@office"})
	var rendered bytes.Buffer
	renderSelector(&rendered, &state)
	state.model.SetQuery("office")
	renderSelector(&rendered, &state)
	text := rendered.String()
	if !strings.Contains(text, "\x1b[2A") {
		t.Fatalf("redraw = %q, want cursor-up sequence", text)
	}
	lastFrame := text[strings.LastIndex(text, "\x1b[2A"):]
	if strings.Contains(lastFrame, "@home") {
		t.Fatalf("last frame = %q, contains stale filtered option", lastFrame)
	}
}

type selectorRuneReaderStub struct {
	keys []rune
}

func (s *selectorRuneReaderStub) ReadRune() rune {
	if len(s.keys) == 0 {
		return 0
	}
	key := s.keys[0]
	s.keys = s.keys[1:]
	return key
}

func TestCancelOrDieLeavesCancellationErrorsSilent(t *testing.T) {
	s := &session{errw: new(bytes.Buffer)}
	for _, err := range []error{io.EOF, readline.ErrInterrupt, &readline.InterruptError{}} {
		if got := cancelOrDie(s, err); got == nil {
			t.Fatalf("cancelOrDie(%T) = nil, want failure", err)
		}
	}
	if s.errw.(*bytes.Buffer).Len() != 0 {
		t.Fatal("cancellation wrote stderr")
	}
	if got := cancelOrDie(s, errors.New("bad input")); got == nil {
		t.Fatal("cancelOrDie(non-cancellation) = nil, want failure")
	}
	if s.errw.(*bytes.Buffer).String() != "bad input\n" {
		t.Fatalf("non-cancellation stderr = %q, want error text", s.errw.(*bytes.Buffer).String())
	}
}
