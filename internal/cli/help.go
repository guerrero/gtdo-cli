package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
)

// The one-line descriptions of the two always-registered actions. Both are
// used as the shorthelp list line and as the help block text. todo.sh's
// equivalents describe functionality out of gtdo's scope; gtdo has none.
const (
	shorthelpText = "List the one-line usage of all built-in actions."
	helpText      = "Display help about usage, options and built-in actions, or just the usage help for the passed ACTION(s)."
)

// optionsHelp is the static Options section of the full help. The flags are
// parsed by the pre-parser, not by cobra, so they are described by hand.
// The wording follows todo.sh's help() wherever the behavior is shared;
// -V and -x describe gtdo's actual behavior (build info, no-op) instead.
const optionsHelp = `  Options:
    -@
        Hide context names in list output.  Use twice to show context names (default).
    -+
        Hide project names in list output.  Use twice to show project names (default).
    -c
        Color mode
    -d CONFIG_FILE
        Use a configuration file other than one of the defaults
    -f
        Forces actions without confirmation or interactive input
    -h
        Display a short help message; same as action "shorthelp"
    -p
        Plain mode turns off colors
    -P
        Hide priority labels in list output.  Use twice to show priority labels (default).
    -a
        Don't auto-archive tasks automatically on completion
    -A
        Auto-archive tasks automatically on completion
    -n
        Don't preserve line numbers; automatically remove blank lines on task deletion
    -N
        Preserve line numbers
    -t
        Prepend the current date to a task automatically when it's added.
    -T
        Do not prepend the current date to a task automatically when it's added.
    -v
        Verbose mode turns on confirmation messages
    -vv
        Extra verbose mode prints some debugging information and additional help text
    -V
        Displays version and build information
    -x
        No-op; accepted for compatibility with todo.txt-cli

`

// envVarsHelp is the Environment variables section of the extra-verbose
// help (-vv): the "additional help text" the -vv option line promises.
// It lists gtdo's own env vars (§5.2 of the design plan) in the shape of
// todo.sh's section, with gtdo's own wording. There is deliberately no
// TODOTXT_* var that gtdo does not implement (§2): gtdo's
// config file env var is GTDO_CONFIG, listed under -d in optionsHelp.
// The name column is padded to 31 characters, then two spaces.
const envVarsHelp = `  Environment variables:
    TODO_DIR                         path to the directory that holds the todo files
    TODO_FILE                        path to the todo.txt file
    DONE_FILE                        path to the done.txt file
    REPORT_FILE                      path to the report.txt file
    TODOTXT_AUTO_ARCHIVE             is same as option -a (0)/-A (1)
    TODOTXT_FORCE=1                  is same as option -f
    TODOTXT_PRESERVE_LINE_NUMBERS    is same as option -n (0)/-N (1)
    TODOTXT_PLAIN                    is same as option -p (1)/-c (0)
    TODOTXT_DATE_ON_ADD              is same as option -t (1)/-T (0)
    TODOTXT_PRIORITY_ON_ADD=pri      default priority A-Z
    TODOTXT_VERBOSE=1                is same as option -v
    TODOTXT_DEFAULT_ACTION=""        run this when called with no arguments
    TODOTXT_SOURCEVAR=$DONE_FILE     use another source for listcon, listproj
    SENTENCE_DELIMITERS=,.:;         suppress the added space in append
`

// builtinActionsHeader closes the options (and optional environment
// variables) part of the help; the per-action blocks follow it.
const builtinActionsHeader = "  Built-in Actions:\n"

// HelpString renders gtdo's full help (§6.4): the usage line, the options,
// the environment-variables section when verbosity is 2+ (the -vv option
// line promises it; todo.sh gates the same section on TODOTXT_VERBOSE > 1),
// and the per-action block of every registered command, in that order.
func HelpString(root *cobra.Command, cfg *config.Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Usage: %s\n\n", onelineUsage())
	b.WriteString(optionsHelp)
	if cfg.Verbose > 1 {
		b.WriteString(envVarsHelp)
	}
	b.WriteString(builtinActionsHeader)
	for _, cmd := range actionCommands(root) {
		b.WriteString(actionBlock(cmd, cmd.Name()))
		b.WriteString("\n\n")
	}
	return b.String()
}

// actionCommands returns the root's action commands in display order (cobra
// sorts by name), excluding hidden commands: cobra adds the hidden
// __complete helper to every root during Execute, and it must never surface
// in gtdo's own texts.
func actionCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Hidden {
			continue
		}
		out = append(out, cmd)
	}
	return out
}

// actionBlock renders the help block for one action: its usage line(s)
// indented four spaces and its Long description indented six, in the shape
// of todo.sh's actionsHelp output. Querying the action by name shows every
// alias usage line; querying it by one alias shows only that line, like
// todo.sh's actionUsage sed extraction.
func actionBlock(cmd *cobra.Command, name string) string {
	var b strings.Builder
	for _, line := range actionUsageLines(cmd, name) {
		fmt.Fprintf(&b, "    %s\n", line)
	}
	if cmd.Long != "" {
		for _, line := range strings.Split(strings.TrimSuffix(cmd.Long, "\n"), "\n") {
			fmt.Fprintf(&b, "      %s\n", line)
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// actionUsageLines returns the usage lines of an action: the command's Use
// line, plus one line per alias with the alias in place of the command
// name. When the query matched an alias, only that alias's line is returned.
func actionUsageLines(cmd *cobra.Command, name string) []string {
	aliasLine := func(alias string) string {
		return strings.Replace(cmd.Use, cmd.Name(), alias, 1)
	}
	if name != cmd.Name() {
		return []string{aliasLine(name)}
	}
	lines := []string{cmd.Use}
	for _, alias := range cmd.Aliases {
		lines = append(lines, aliasLine(alias))
	}
	return lines
}
