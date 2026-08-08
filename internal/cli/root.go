// Package cli defines gtdo's cobra command tree, the getopts-style flag
// pre-parser, and the entry point shared by cmd/gtdo/main.go and the
// testscript harness.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
)

// NewRootCmd builds the gtdo command tree. Actions are wired in later tasks;
// the root itself handles no-action and unknown-action invocations via the
// cobra "unknown command" error and RunE.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:                "gtdo",
		DisableFlagParsing: true, // the pre-parser owns all flag parsing
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Unreachable for known actions; only reached when cobra ran the
			// root with leftover args, which never happens (see Execute).
			return nil
		},
	}
	// shorthelp stub: the pre-parser rewrites -h to the shorthelp action.
	// The full action list arrives in Task 11; for now it prints the usage
	// line so -h exits 0 with output, like todo.sh.
	root.AddCommand(&cobra.Command{
		Use:    "shorthelp",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), ShorthelpString())
			return nil
		},
	})
	return root
}

// Execute resolves configuration, prepares the store, and dispatches through
// cobra with the pre-parsed action. It is the single dispatch path for both
// the real binary and the tests.
func Execute(root *cobra.Command, opts *config.Options, action string, rest []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if action == "" {
		// Default action arrives in Task 11; for now, no action is usage.
		fmt.Fprint(stdout, UsageString())
		return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}
	// Unknown action → usage, exactly like todo.sh's `*) usage;;`.
	// Cobra only reports "unknown command" once subcommands exist (Task 6);
	// the explicit check keeps the empty-tree behavior identical.
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

// Run is the shared entry point: pre-parse, version, dispatch, exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, action, rest, err := Preparse(args)
	if err != nil {
		fmt.Fprint(stdout, UsageString())
		return exitcode.Generic
	}
	if opts.Version {
		fmt.Fprint(stdout, VersionString())
		return exitcode.OK
	}
	root := NewRootCmd()
	if err := Execute(root, opts, action, rest, stdin, stdout, stderr); err != nil {
		// Cobra reports unknown commands ("unknown command \"bogus\"...");
		// todo.sh prints the usage text instead, on stdout, exit 1.
		if strings.HasPrefix(err.Error(), "unknown command ") {
			fmt.Fprint(stdout, UsageString())
			return exitcode.Generic
		}
		return exitcode.Of(err)
	}
	return exitcode.OK
}
