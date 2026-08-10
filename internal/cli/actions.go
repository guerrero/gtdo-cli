package cli

// The mutating actions (plan §6.3): add/addm/addto, append, prepend,
// replace, pri, depri, do, del, move, archive, and report, wired to
// internal/todo with todo.sh-exact messages, prompts, and exit codes.
//
// Each action follows the same shape as its todo.sh case block: validate
// and fetch the task first (getTodo), prompt or take the input, mutate
// the file through the Store, then print the verbose messages. Messages
// are byte-identical to todo.sh: die texts go to stderr with exit 1,
// confirmations to stdout when verbose (the default).

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
	"github.com/guerrero/gtdo/internal/todo"
)

// registerActions adds the §6.3 actions to root. The description texts
// follow todo.sh's actionsHelp output (minus the out-of-scope
// references).
func registerActions(root *cobra.Command, cfg *config.Config) {
	add := func(spec actionSpec) { root.AddCommand(newAction(spec, cfg)) }
	add(actionSpec{
		use:     `add "THING I NEED TO DO +project @context"`,
		aliases: []string{"a"},
		short:   "Add a TODO item to todo.txt.",
		long:    "Adds THING I NEED TO DO to your todo.txt file on its own line.\nProject and context notation optional.\nQuotes optional.",
		run:     actionAdd,
	})
	add(actionSpec{
		use:   `addm "THINGS I NEED TO DO"`,
		short: "Add multiple TODO items to todo.txt.",
		long:  "Adds each line of THINGS I NEED TO DO to your todo.txt on its own line.\nProject and context notation optional.",
		run:   actionAddm,
	})
	add(actionSpec{
		use:   `addto DEST "TEXT TO ADD"`,
		short: "Add a line of text to a file in the todo.txt directory.",
		long:  "Adds a line of text to any file located in the todo.txt directory.\nFor example, addto inbox.txt \"decide about vacation\"",
		run:   actionAddto,
	})
	add(actionSpec{
		use:       `append NR "TEXT TO APPEND"`,
		aliases:   []string{"app"},
		short:     "Append text to the task on line NR.",
		validArgs: firstArgNumbers(cfg),
		long:      "Adds TEXT TO APPEND to the end of the task on line NR.\nQuotes optional.",
		run:       actionAppend,
	})
	add(actionSpec{
		use:   "archive",
		short: "Move done tasks from todo.txt to done.txt and remove blank lines.",
		long:  "Moves all done tasks from todo.txt to done.txt and removes blank lines.",
		run:   actionArchive,
	})
	add(actionSpec{
		use:       "del NR [TERM]",
		aliases:   []string{"rm"},
		short:     "Delete the task on line NR, or remove TERM from it.",
		validArgs: firstArgNumbers(cfg),
		long:      "Deletes the task on line NR in todo.txt.\nIf TERM specified, deletes only TERM from the task.",
		run:       actionDel,
	})
	add(actionSpec{
		use:       "depri NR [NR ...]",
		aliases:   []string{"dp"},
		short:     "Remove the priority from the task(s) on line NR.",
		validArgs: everyArgNumbers(cfg),
		long:      "Deprioritizes (removes the priority) from the task(s) on line NR in todo.txt.",
		run:       actionDepri,
	})
	add(actionSpec{
		use:       "do NR [NR ...]",
		aliases:   []string{"done"},
		short:     "Mark task(s) on line NR as done.",
		validArgs: everyArgNumbers(cfg),
		long:      "Marks task(s) on line NR as done in todo.txt.",
		run:       actionDo,
	})
	add(actionSpec{
		use:       "move NR DEST [SRC]",
		aliases:   []string{"mv"},
		short:     "Move a task from one file to another in the todo.txt directory.",
		validArgs: firstArgNumbers(cfg),
		long:      "Moves the line NR from source text file (SRC) to destination text file (DEST).\nBoth source and destination file must be located in the directory defined in the\nconfiguration directory.  When SRC is not defined it's by default todo.txt.",
		run:       actionMove,
	})
	add(actionSpec{
		use:       `prepend NR "TEXT TO PREPEND"`,
		aliases:   []string{"prep"},
		short:     "Prepend text to the task on line NR.",
		validArgs: firstArgNumbers(cfg),
		long:      "Adds TEXT TO PREPEND to the beginning of the task on line NR.\nQuotes optional.",
		run:       actionPrepend,
	})
	add(actionSpec{
		use:       "pri NR PRIORITY [NR PRIORITY ...]",
		aliases:   []string{"p"},
		short:     "Add or replace the priority of the task on line NR.",
		validArgs: evenArgNumbers(cfg),
		long:      "Adds PRIORITY to task on line NR.  If the task is already prioritized,\nreplaces current priority with new PRIORITY.\nPRIORITY must be a letter between A and Z.",
		run:       actionPri,
	})
	add(actionSpec{
		use:       `replace NR "UPDATED TODO"`,
		short:     "Replace the task on line NR.",
		validArgs: firstArgNumbers(cfg),
		long:      "Replaces task on line NR with UPDATED TODO.",
		run:       actionReplace,
	})
	add(actionSpec{
		use:   "report",
		short: "Add the number of open and done tasks to report.txt.",
		long:  "Adds the number of open tasks and done tasks to report.txt.",
		run:   actionReport,
	})
}

// session bundles the resolved configuration, the prepared store, and the
// I/O streams of one action invocation.
type session struct {
	cfg   *config.Config
	store *todo.Store
	in    io.Reader
	out   io.Writer
	errw  io.Writer
}

// newSession prepares the store for one action invocation: the files are
// created on demand, mirroring todo.sh's startup sanity checks (§6.5).
func newSession(cmd *cobra.Command, cfg *config.Config) (*session, error) {
	st := &todo.Store{
		Dir:                 cfg.Dir,
		TodoFile:            cfg.TodoFile,
		DoneFile:            cfg.DoneFile,
		ReportFile:          cfg.ReportFile,
		PreserveLineNumbers: cfg.PreserveLineNumbers,
		EnableUUID:          cfg.EnableUUID,
		SentenceDelimiters:  cfg.SentenceDelimiters,
	}
	if err := st.Ensure(); err != nil {
		// The exact todo.sh die text, contract (§6.5). dieWithHelp's
		// action help preamble is dropped: only help/shorthelp trigger it,
		// and the message itself is the observable contract.
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return nil, exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}
	return &session{
		cfg:   cfg,
		store: st,
		in:    cmd.InOrStdin(),
		out:   cmd.OutOrStdout(),
		errw:  cmd.ErrOrStderr(),
	}, nil
}

// die mirrors todo.sh's die: message to stderr, exit 1. The caller has
// already done the work the message reports; nothing else is printed.
func (s *session) die(msg string) error {
	fmt.Fprintln(s.errw, msg)
	return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
}

// verbose mirrors `[ "$TODOTXT_VERBOSE" -gt 0 ]`.
func (s *session) verbose() bool { return s.cfg.Verbose > 0 }

// parseItem validates a task-number argument the way todo.sh's getTodo
// does: empty or non-numeric dies with the action's usage message.
func parseItem(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}

// splitItems splits space- and comma-separated task numbers, todo.sh's
// `${*//,/ }` word split used by do and depri.
func splitItems(args []string) []string {
	var items []string
	for _, arg := range args {
		for _, item := range strings.Split(arg, ",") {
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

// stdinIsTTY reports whether r is connected to an interactive terminal.
// bash's read -p prints its prompt only when stdin is a terminal; the
// txtar harness feeds an in-memory reader, so the prompts must stay
// silent there (the shell tests pipe stdin and expect no prompt either).
func stdinIsTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// prompt writes p to stderr when stdin is a terminal, where bash's read -p
// puts it. Non-terminal stdin reads silently.
func (s *session) prompt(p string) {
	if stdinIsTTY(s.in) {
		fmt.Fprint(s.errw, p)
	}
}

// readLine reads one line of stdin like `read -r`: the newline is
// stripped, backslashes are literal, and EOF yields the empty string.
func (s *session) readLine() string {
	line, err := bufio.NewReader(s.in).ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSuffix(line, "\n")
}

// confirm asks a y/n question, todo.sh's confirm(): with -f it answers
// yes without asking; otherwise it prints the question (TTY only), reads
// one character, echoes a newline, and returns true for 'y'. The single
// character read matches modern bash's `read -N 1`; an Enter after the
// answer is left unconsumed, which the shell tests never rely on.
func (s *session) confirm(question string) (bool, error) {
	if s.cfg.Force {
		return true, nil
	}
	s.prompt(question + "? (y/n) ")
	var buf [1]byte
	n, err := io.ReadFull(s.in, buf[:])
	answer := err == nil && n == 1 && buf[0] == 'y'
	fmt.Fprintln(s.out)
	return answer, nil
}

// isRegular mirrors todo.sh's `[ -f path ]` for the CLI's move checks.
func isRegular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// now returns the current time, unless the test harness pinned it with
// GTDO_TEST_NOW (RFC 3339). The shell suite fakes date with a bin/date
// shim over TODO_TEST_TIME (2009-02-13T04:40:00Z); gtdo's txtar harness
// sets the env var instead, which scripts override to advance the clock.
func now() time.Time {
	if v := os.Getenv("GTDO_TEST_NOW"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Now()
}
