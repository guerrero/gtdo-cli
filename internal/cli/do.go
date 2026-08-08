package cli

// The do action (§6.3): marks tasks done and auto-archives them. The
// already-done complaint goes to stderr and makes the exit status 1; the
// archive runs afterwards regardless, exactly like todo.sh's recursive
// "$TODO_FULL_SH archive" invocation.

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
)

// actionDo marks each item done by prepending "x YYYY-MM-DD " (removing
// an existing priority), then runs the archive action when auto_archive
// is on. Items are processed one at a time like todo.sh's loop.
func actionDo(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	usage := `usage: ` + ProgName + ` do NR [NR ...]`
	if len(args) == 0 {
		return s.die(usage)
	}
	status := exitcode.OK
	for _, raw := range splitItems(args) {
		n, ok := parseItem(raw)
		if !ok {
			return s.die(usage)
		}
		results, err := s.store.Do([]int{n}, now())
		for _, res := range results {
			if res.AlreadyDone {
				fmt.Fprintf(s.errw, "TODO: %d is already marked done.\n", res.LineNumber)
				status = exitcode.Generic
			} else if s.verbose() {
				fmt.Fprintf(s.out, "%d %s\n", res.LineNumber, res.NewText)
				fmt.Fprintf(s.out, "TODO: %d marked as done.\n", res.LineNumber)
			}
		}
		if err != nil {
			return s.die(err.Error())
		}
	}
	if cfg.AutoArchive {
		if err := runArchive(s); err != nil {
			// todo.sh: "$TODO_FULL_SH" archive || status=$? — the
			// child's die message is already printed; record the status.
			fmt.Fprintln(s.errw, err)
			status = exitcode.Generic
		}
	}
	if status != exitcode.OK {
		return exitcode.Wrap(status, exitcode.ErrFailure)
	}
	return nil
}
