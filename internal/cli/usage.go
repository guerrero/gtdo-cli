package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ProgName is the program name used in usage and help texts, mirroring
// todo.sh's TODO_SH (basename "$0"). main sets it from os.Args[0]; the test
// harness leaves the default.
var ProgName = "gtdo"

// onelineUsage is the shared one-line usage string (§6.4). Its flag set is
// the plan's literal string even though the full flag set is larger.
func onelineUsage() string {
	return fmt.Sprintf("%s [-fhpantvV] [-d todo_config] action [task_number] [task_description]", ProgName)
}

// UsageString is the usage text printed to stdout when no action (or an
// unknown action/flag) is given; exit status 1. Spec §6.4.
func UsageString() string {
	return fmt.Sprintf("Usage: %s\nTry '%s -h' for more information.\n", onelineUsage(), ProgName)
}

// ShorthelpString renders the -h/shorthelp text (§6.4): the usage line, the
// one-line action list from the registered commands, and the closing hint.
// The list covers only gtdo's own actions.
func ShorthelpString(root *cobra.Command) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  Usage: %s\n\n  Actions:\n", onelineUsage())
	for _, cmd := range actionCommands(root) {
		fmt.Fprintf(&b, "    %s  %s\n", cmd.Name(), cmd.Short)
	}
	fmt.Fprintf(&b, "\n  See \"help\" for more details.\n")
	return b.String()
}
