package cli

// The add family (§6.3): add, addm, and addto. All three append a task
// through the store and print the _addto confirmation — "N text" and
// "PREFIX: N added." — with the prefix from the destination file's name.

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
	"github.com/guerrero/gtdo/internal/todo"
)

// actionAdd appends one task to todo.txt. Without an argument it prompts
// "Add: " (unless -f), reading a line of stdin like todo.sh's
// `read -p "Add: " -e -r input`.
func actionAdd(cmd *cobra.Command, args []string, cfg *config.Config) error {
	opts, err := parseAddOptions(args)
	if err != nil || (cfg.Force && opts.Mode != addModeNone) {
		fmt.Fprintln(cmd.ErrOrStderr(), addUsage())
		return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}

	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	if opts.Mode == addModeNone {
		return addLegacy(s, opts.Positional)
	}

	candidates := collectAddCandidates(cfg)
	input := newAddInput(s, candidates)
	if closer, ok := input.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}
	var text string
	if opts.Mode == addModeInteractive {
		text, err = input.PromptTask(candidates)
	} else {
		text, err = runGuided(input, candidates, opts)
	}
	if err != nil {
		return cancelOrDie(s, err)
	}
	line, text, err := s.store.Add(text, cfg.DateOnAdd, cfg.PriorityOnAdd, now())
	if err != nil {
		return s.die(err.Error())
	}
	return s.printAdded(s.store.TodoFile, line, text)
}

// addLegacy keeps the original add path intact for positional invocations.
// Its prompt, usage, Store.Add call, and confirmation output are deliberately
// unchanged so introducing explicit modes cannot alter the byte contract.
func addLegacy(s *session, args []string) error {
	input, err := s.addInput(args, `usage: `+ProgName+` add "TODO ITEM"`)
	if err != nil {
		return err
	}
	line, text, err := s.store.Add(input, s.cfg.DateOnAdd, s.cfg.PriorityOnAdd, now())
	if err != nil {
		return s.die(err.Error())
	}
	return s.printAdded(s.store.TodoFile, line, text)
}

// cancelOrDie maps explicit-mode cancellation to a silent generic failure.
// Readline returns ErrInterrupt from Instance.Readline while lower-level
// terminal tests and adapters may expose InterruptError directly; both forms
// represent the same no-write cancellation contract.
func cancelOrDie(s *session, err error) error {
	if err == nil {
		return nil
	}
	var interrupt *readline.InterruptError
	if errors.Is(err, io.EOF) || errors.Is(err, readline.ErrInterrupt) || errors.As(err, &interrupt) {
		return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}
	return s.die(err.Error())
}

// actionAddm adds each non-empty line of the input as its own task,
// printing one _addto confirmation per line (t2000 'actual multiline
// add').
func actionAddm(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	input, err := s.addInput(args, `usage: `+ProgName+` addm "TODO ITEM"`)
	if err != nil {
		return err
	}
	results, err := s.store.Addm(input, cfg.DateOnAdd, cfg.PriorityOnAdd, now())
	if err != nil {
		return s.die(err.Error())
	}
	for _, res := range results {
		if err := s.printAdded(s.store.TodoFile, res.LineNumber, res.Text); err != nil {
			return err
		}
	}
	return nil
}

// actionAddto appends one task to DEST inside the todo directory. The
// destination must already exist; addto never prompts, -f or not.
func actionAddto(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	if len(args) < 2 || args[1] == "" {
		return s.die(`usage: ` + ProgName + ` addto DEST "TODO ITEM"`)
	}
	input := strings.Join(args[1:], " ")
	line, text, err := s.store.Addto(args[0], input, cfg.DateOnAdd, cfg.PriorityOnAdd, now())
	if err != nil {
		return s.die(err.Error())
	}
	return s.printAdded(filepath.Join(s.store.Dir, args[0]), line, text)
}

// addInput mirrors the `if [[ -z "$2" && $TODOTXT_FORCE = 0 ]]` branch of
// add/addm: prompt when the argument is missing and -f is off, otherwise
// join the arguments with spaces (and die with usage when -f has nothing
// to add).
func (s *session) addInput(args []string, usage string) (string, error) {
	if len(args) == 0 && !s.cfg.Force {
		s.prompt("Add: ")
		return s.readLine(), nil
	}
	if len(args) == 0 {
		return "", s.die(usage)
	}
	return strings.Join(args, " "), nil
}

// printAdded prints the _addto verbose confirmation: "N text" and
// "PREFIX: N added.".
func (s *session) printAdded(path string, line int, text string) error {
	if !s.verbose() {
		return nil
	}
	fmt.Fprintf(s.out, "%d %s\n", line, text)
	fmt.Fprintf(s.out, "%s: %d added.\n", todo.Prefix(path), line)
	return nil
}
