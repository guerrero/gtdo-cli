package cli

import "fmt"

// ProgName is the program name used in usage and help texts, mirroring
// todo.sh's TODO_SH (basename "$0"). main sets it from os.Args[0]; the test
// harness leaves the default.
var ProgName = "gtdo"

// UsageString is the usage text printed to stdout when no action (or an
// unknown action/flag) is given; exit status 1. Spec §6.4.
func UsageString() string {
	return fmt.Sprintf("Usage: %s [-fhpantvV] [-d todo_config] action [task_number] [task_description]\n"+
		"Try '%s -h' for more information.\n", ProgName, ProgName)
}

// ShorthelpString is the -h/shorthelp text. The full action list arrives in
// Task 11; for now it carries the usage line so -h exits 0 with output.
func ShorthelpString() string {
	return fmt.Sprintf("  Usage: %s [-fhpantvV] [-d todo_config] action [task_number] [task_description]\n", ProgName)
}
