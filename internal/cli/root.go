// Package cli defines gtdo's cobra command tree, the getopts-style flag
// pre-parser, and the entry point shared by cmd/gtdo/main.go and the
// testscript harness.
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
)

// NewRootCmd builds the gtdo command tree. The actions of Tasks 7-8 are
// registered through newAction, the same constructor help and shorthelp use.
// No-action and unknown-action invocations never reach cobra: Execute checks
// the action against the tree first and prints the usage text itself.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use: "gtdo",
		// The pre-parser owns all flag parsing (§6.1): flags are accepted
		// only before the action, and cobra must never see -h (which the
		// pre-parser rewrites to the shorthelp action) or --help (which is
		// an unknown flag and a usage error).
		DisableFlagParsing: true,
		// Usage and error texts are gtdo's own (§6.4); cobra prints
		// neither, not even for errors returned by action RunEs.
		SilenceUsage:  true,
		SilenceErrors: true,
		// The default completion command is cobra's, not gtdo's; Task 10
		// wires completion deliberately.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	shorthelp := newAction(actionSpec{
		use:   "shorthelp",
		short: shorthelpText,
		long:  shorthelpText,
		run: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprint(cmd.OutOrStdout(), ShorthelpString(root))
			return nil
		},
	})
	root.AddCommand(shorthelp)

	// help is registered as the help command itself: cobra's InitDefaultHelpCmd
	// would otherwise add its own "help" command (with cobra's help template)
	// alongside ours. SetHelpCommand adopts ours, so no second help command
	// ever appears.
	help := newAction(actionSpec{
		use:   "help [ACTION...]",
		short: helpText,
		long:  helpText,
		run: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprint(cmd.OutOrStdout(), HelpString(root))
				return nil
			}
			for _, name := range args {
				child, _, _ := root.Find([]string{name})
				if child == root || child.Hidden {
					// todo.sh: die "TODO: No action \"$1\" exists." —
					// stderr, exit 1, nothing else printed.
					fmt.Fprintf(cmd.ErrOrStderr(), "TODO: No action %q exists.\n", name)
					return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", actionBlock(child, name))
			}
			return nil
		},
	})
	root.SetHelpCommand(help)
	root.AddCommand(help)

	return root
}

// actionSpec is the registration shape for a gtdo action command. help and
// shorthelp are built from it, and Tasks 7-8 register the todo.sh actions
// the same way.
type actionSpec struct {
	use     string   // cobra Use line, e.g. `add "THING I NEED TO DO +project @context"`
	aliases []string // todo.sh aliases, e.g. ["a"]; extra usage lines in help blocks
	short   string   // one-line description for shorthelp's action list
	long    string   // block description for help [ACTION]
	run     func(cmd *cobra.Command, args []string) error
}

// newAction turns an actionSpec into a cobra command with gtdo's shared
// settings: the pre-parser owns flags, and cobra never prints its own usage
// or error text — gtdo's messages and exit codes are the parity contract.
func newAction(spec actionSpec) *cobra.Command {
	return &cobra.Command{
		Use:                spec.use,
		Aliases:            spec.aliases,
		Short:              spec.short,
		Long:               spec.long,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE:               spec.run,
	}
}

// Execute dispatches the pre-parsed action through cobra. It is the single
// dispatch path for both the real binary and the tests: an unknown action
// prints the usage text to stdout and fails with exit code 1, exactly like
// todo.sh's `*) usage;;` case.
func Execute(root *cobra.Command, action string, rest []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if child, _, _ := root.Find([]string{action}); child == root {
		fmt.Fprint(stdout, UsageString())
		return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}
	root.SetArgs(append([]string{action}, rest...))
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	return root.Execute()
}

// Run is the shared entry point: pre-parse, version, config resolution,
// default action, and dispatch. It returns the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, action, rest, err := Preparse(args)
	if err != nil {
		fmt.Fprint(stdout, UsageString())
		return exitcode.Generic
	}
	if opts.Version {
		// -V exits before config resolution, like todo.sh's version().
		fmt.Fprint(stdout, VersionString())
		return exitcode.OK
	}
	cfg, err := config.Load(*opts)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", ProgName, err)
		return exitcode.Generic
	}
	if action == "" {
		// §6.5: with no action given, act as if the configured default
		// action was typed; an empty default is a usage error.
		action = cfg.DefaultAction
	}
	if action == "" {
		fmt.Fprint(stdout, UsageString())
		return exitcode.Generic
	}
	root := NewRootCmd()
	if err := Execute(root, action, rest, stdin, stdout, stderr); err != nil {
		return exitcode.Of(err)
	}
	return exitcode.OK
}
