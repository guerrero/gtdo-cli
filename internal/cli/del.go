package cli

// The del action (§6.3): with no TERM it asks for confirmation (unless
// -f) and deletes the task; with TERM it removes the term from the task
// text via the five sed rules of todo.sh. Both variants print the
// verbose confirmations.

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
)

// actionDel handles `del NR` and `del NR TERM`.
func actionDel(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	usage := `usage: ` + ProgName + ` del NR [TERM]`
	if len(args) < 1 {
		return s.die(usage)
	}
	item, ok := parseItem(args[0])
	if !ok {
		return s.die(usage)
	}
	if len(args) >= 2 {
		return s.delTerm(item, args[1])
	}
	task, err := s.store.TaskAt(s.store.TodoFile, item)
	if err != nil {
		return s.die(err.Error())
	}
	yes, err := s.confirm(fmt.Sprintf("Delete '%s'", task.Text))
	if err != nil {
		return err
	}
	if !yes {
		return s.die("TODO: No tasks were deleted.")
	}
	removed, err := s.store.Del(item)
	if err != nil {
		return s.die(err.Error())
	}
	if s.verbose() {
		fmt.Fprintf(s.out, "%d %s\n", item, removed)
		fmt.Fprintf(s.out, "TODO: %d deleted.\n", item)
	}
	return nil
}

// delTerm removes term from the item-th task, todo.sh's five sed rules in
// order (del.go's DelTerm). A term that changes nothing dies with the
// not-found text; a term that blanks the whole task dies with getNewtodo's
// text after the line is already blanked.
func (s *session) delTerm(item int, term string) error {
	oldText, newText, err := s.store.DelTerm(item, term)
	if err != nil {
		if newText == oldText && s.verbose() {
			// The not-found die prints the unchanged task first.
			fmt.Fprintf(s.out, "%d %s\n", item, oldText)
		}
		return s.die(err.Error())
	}
	if s.verbose() {
		fmt.Fprintf(s.out, "%d %s\n", item, oldText)
		fmt.Fprintf(s.out, "TODO: Removed '%s' from task.\n", term)
		fmt.Fprintf(s.out, "%d %s\n", item, newText)
	}
	return nil
}
