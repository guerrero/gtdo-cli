package cli

// The archive and report actions (§6.3). archive moves the "x " lines to
// done.txt and defragments blank lines; report archives first and then
// appends or reuses the count line in report.txt. do and report reuse
// runArchive for todo.sh's recursive "$TODO_FULL_SH archive" call.

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
)

// actionArchive is the archive action: with verbose on it prints the
// archived lines and the archived message, or the no-done-tasks warning.
func actionArchive(cmd *cobra.Command, _ []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	if err := runArchive(s); err != nil {
		return s.die(err.Error())
	}
	return nil
}

// runArchive executes the archive logic shared by the archive action, the
// do auto-archive, and report.
func runArchive(s *session) error {
	archived, err := s.store.Archive()
	if err != nil {
		return err
	}
	if !s.verbose() {
		return nil
	}
	for _, line := range archived {
		fmt.Fprintln(s.out, line)
	}
	if len(archived) > 0 {
		fmt.Fprintf(s.out, "TODO: %s archived.\n", s.store.TodoFile)
	} else {
		fmt.Fprintf(s.out, "TODO: %s does not contain any done tasks.\n", s.store.TodoFile)
	}
	return nil
}

// actionReport appends "TIMESTAMP OPEN DONE" to report.txt (or reuses the
// last line when the counts are unchanged), after archiving first.
func actionReport(cmd *cobra.Command, _ []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	// todo.sh: "$TODO_FULL_SH" archive — the archive runs first and its
	// messages print; its failure does not stop the report.
	if err := runArchive(s); err != nil {
		fmt.Fprintln(s.errw, err)
	}
	line, updated, err := s.store.Report(now())
	if err != nil {
		return s.die(err.Error())
	}
	fmt.Fprintln(s.out, line)
	if s.verbose() {
		if updated {
			fmt.Fprintln(s.out, "TODO: Report file updated.")
		} else {
			fmt.Fprintln(s.out, "TODO: Report file is up-to-date.")
		}
	}
	return nil
}
