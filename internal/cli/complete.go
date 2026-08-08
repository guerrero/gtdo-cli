package cli

// Shell completion (§6.6): the cobra completion command (bash, zsh, fish,
// powershell — all four come free with cobra's default command) and the
// per-action ValidArgsFunctions. gtdo's flags are parsed by the
// pre-parser, never by cobra, so there is nothing to complete for them;
// the actions complete their positional arguments instead.
//
// The candidates mirror the task numbers `list` shows and the words
// listcon/listproj list, read from the current TODO_FILE. Reading is
// best-effort: a missing or unreadable TODO_FILE yields no completions,
// because shell completion must never fail the command or create files.

import (
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/todo"
)

// firstArgNumbers completes task numbers at the first argument position:
// the NR slot of append, del, move, prepend, and replace.
func firstArgNumbers(cfg *config.Config) cobra.CompletionFunc {
	return positionNumbers(cfg, func(n int) bool { return n == 0 })
}

// everyArgNumbers completes task numbers at every argument position: do
// and depri take NR [NR ...].
func everyArgNumbers(cfg *config.Config) cobra.CompletionFunc {
	return positionNumbers(cfg, func(int) bool { return true })
}

// evenArgNumbers completes task numbers at the even argument positions:
// pri's NR PRIORITY [NR PRIORITY ...] pairs.
func evenArgNumbers(cfg *config.Config) cobra.CompletionFunc {
	return positionNumbers(cfg, func(n int) bool { return n%2 == 0 })
}

// positionNumbers completes task numbers when the argument position is
// one of the task-number slots, and nothing otherwise.
func positionNumbers(cfg *config.Config, at func(int) bool) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if !at(len(args)) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return taskNumbers(cfg.TodoFile, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// taskNumbers returns the line numbers the NR-taking actions accept — the
// real line numbers `list` shows, i.e. every non-blank, non-digit-only
// line of path — filtered by the typed prefix. A missing or unreadable
// file yields no completions.
func taskNumbers(path, toComplete string) []cobra.Completion {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []cobra.Completion
	for i, line := range strings.Split(string(text), "\n") {
		if !listableLine(line) {
			continue
		}
		n := strconv.Itoa(i + 1)
		if strings.HasPrefix(n, toComplete) {
			out = append(out, n)
		}
	}
	return out
}

// listableLine reports whether the numbering keeps the line: blank lines
// and lines of only digits and spaces are dropped (the numbering sed's
// `/^[ 0-9]\{1,\} *$/d`, matched here as `^[ 0-9]*$` like Format).
func listableLine(line string) bool {
	for _, c := range line {
		if c != ' ' && (c < '0' || c > '9') {
			return true
		}
	}
	return false
}

// termCompletions completes the list/listall term arguments with the
// @contexts and +projects of TODO_FILE — exactly the words listcon and
// listproj would list. The typed sigil picks the kind; with neither, both
// are offered.
func termCompletions(cfg *config.Config) cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		text, err := os.ReadFile(cfg.TodoFile)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		lines := strings.Split(string(text), "\n")
		var out []cobra.Completion
		for _, sigil := range []byte{'@', '+'} {
			if toComplete != "" && toComplete[0] != sigil {
				continue
			}
			words, err := todo.SigilWords(lines, sigil, nil)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			for _, w := range words {
				if strings.HasPrefix(w, toComplete) {
					out = append(out, w)
				}
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
