// Package script contains helper commands used by the CLI testscript suite.
package script

import (
	"errors"
	"os"
	"os/exec"
)

// Main runs its target and exits with status 1 only when the target itself
// exits with status 1. Returning success for every other target status lets a
// negated testscript exec fail, pinning the expected status without a shell.
func Main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}

	command := exec.Command(os.Args[1], os.Args[2:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		os.Exit(1)
	}
}
