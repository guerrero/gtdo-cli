package cli

// The text-editing actions (§6.3): append, prepend, and replace. They
// share todo.sh's replaceOrPrepend shape: validate the task first
// (getTodo), prompt or take the input, mutate, then print the verbose
// confirmation.

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/todo"
)

// actionAppend appends text to the item-th task. Without text it prompts
// "Append: " (unless -f).
func actionAppend(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	usage := `usage: ` + ProgName + ` append NR "TEXT TO APPEND"`
	if len(args) < 1 {
		return s.die(usage)
	}
	item, ok := parseItem(args[0])
	if !ok {
		return s.die(usage)
	}
	if _, err := s.store.TaskAt(s.store.TodoFile, item); err != nil {
		return s.die(err.Error())
	}
	var input string
	if len(args) < 2 && !cfg.Force {
		s.prompt("Append: ")
		input = s.readLine()
	} else {
		input = strings.Join(args[1:], " ")
	}
	newText, err := s.store.Append(item, input)
	if err != nil {
		return s.die(err.Error())
	}
	if s.verbose() {
		fmt.Fprintf(s.out, "%d %s\n", item, newText)
	}
	return nil
}

// actionPrepend inserts text at the start of the item-th task, keeping
// its priority and date prefix. Without text it prompts "Prepend: "
// (unless -f).
func actionPrepend(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	return s.replaceOrPrepend(cfg, args, `usage: `+ProgName+` prepend NR "TEXT TO PREPEND"`, "Prepend: ", false)
}

// actionReplace swaps the item-th task's text. Without text it prompts
// "Replacement: " (unless -f).
func actionReplace(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	return s.replaceOrPrepend(cfg, args, `usage: `+ProgName+` replace NR "UPDATED ITEM"`, "Replacement: ", true)
}

// replaceOrPrepend is todo.sh's replaceOrPrepend: getTodo, prompt or take
// the input, mutate, and print the verbose confirmation — replace prints
// old and new text around a "TODO: Replaced task with:" line, prepend
// only the new text.
func (s *session) replaceOrPrepend(cfg *config.Config, args []string, usage, querytext string, isReplace bool) error {
	if len(args) < 1 {
		return s.die(usage)
	}
	item, ok := parseItem(args[0])
	if !ok {
		return s.die(usage)
	}
	if _, err := s.store.TaskAt(s.store.TodoFile, item); err != nil {
		return s.die(err.Error())
	}
	var input string
	if len(args) < 2 && !cfg.Force {
		s.prompt(querytext)
		input = s.readLine()
	} else {
		input = strings.Join(args[1:], " ")
	}

	var oldText, newText string
	var err error
	if isReplace {
		oldText, newText, err = s.store.Replace(item, input)
	} else {
		newText, err = s.store.Prepend(item, input)
	}
	if err != nil {
		return s.die(err.Error())
	}
	if newText == "" {
		// getNewtodo: the replacement blanked the whole line, and the
		// file is already written — die with the exact text.
		return s.die(fmt.Sprintf("%s: No updated task %d.", todo.Prefix(s.store.TodoFile), item))
	}
	if !s.verbose() {
		return nil
	}
	if isReplace {
		fmt.Fprintf(s.out, "%d %s\n", item, oldText)
		fmt.Fprintln(s.out, "TODO: Replaced task with:")
	}
	fmt.Fprintf(s.out, "%d %s\n", item, newText)
	return nil
}
