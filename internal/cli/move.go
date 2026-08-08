package cli

// The move action (§6.3): moves the item-th task from SRC (default
// todo.txt) to DEST, both inside the todo directory. Confirmation is
// asked unless -f; the verbose confirmation prints the task text and the
// new line number in the destination.

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/todo"
)

// actionMove handles `move NR DEST [SRC]`. The order of the checks
// mirrors todo.sh: usage (missing DEST), source exists, destination
// exists, task exists, then the confirmation.
func actionMove(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	usage := `usage: ` + ProgName + ` mv NR DEST [SRC]`
	if len(args) < 2 {
		return s.die(usage)
	}
	item, ok := parseItem(args[0])
	if !ok {
		return s.die(usage)
	}
	dest := filepath.Join(s.store.Dir, args[1])
	src := s.store.TodoFile
	if len(args) >= 3 {
		src = filepath.Join(s.store.Dir, args[2])
	}
	if !isRegular(src) {
		return s.die(fmt.Sprintf("TODO: Source file %s does not exist.", src))
	}
	if !isRegular(dest) {
		return s.die(fmt.Sprintf("TODO: Destination file %s does not exist.", dest))
	}
	task, err := s.store.TaskAt(src, item)
	if err != nil {
		return s.die(err.Error())
	}
	yes, err := s.confirm(fmt.Sprintf("Move '%s' from %s to %s", task.Text, src, dest))
	if err != nil {
		return err
	}
	if !yes {
		return s.die(todo.Prefix(src) + ": No tasks moved.")
	}
	text, destNum, err := s.store.Move(item, dest, src)
	if err != nil {
		return s.die(err.Error())
	}
	if s.verbose() {
		fmt.Fprintf(s.out, "%d %s\n", item, text)
		fmt.Fprintf(s.out, "%s: %d moved to %d in %s.\n", todo.Prefix(src), item, destNum, todo.Prefix(dest))
	}
	return nil
}
