// Package exitcode defines gtdo's process exit codes and the typed error that
// carries one. Only cmd/gtdo/main.go and the test harness turn these into an
// os.Exit call.
package exitcode

import (
	"errors"
	"fmt"
)

// Process exit codes. todo.sh has exactly two: 0 on success, 1 on any
// failure (die and usage both exit 1). These values are part of the
// byte-parity contract and must never change.
const (
	OK      = 0
	Generic = 1
)

// ErrFailure is the sentinel returned by actions after they have already
// written their own error message to stderr. main must not print anything
// for it — the message is part of the parity contract.
var ErrFailure = errors.New("failure")

// Error is an error annotated with the exit code gtdo should terminate with.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Wrap annotates err with code. It returns nil when err is nil so callers can
// write `return exitcode.Wrap(code, doThing())` unconditionally.
func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

// Of reports the exit code err asks for: OK for nil, the code of the
// outermost *Error in the chain, or Generic when the chain carries no code.
func Of(err error) int {
	if err == nil {
		return OK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return Generic
}

// Wrapf builds a new error with the given code and formatted message.
func Wrapf(code int, format string, a ...any) error {
	return &Error{Code: code, Err: fmt.Errorf(format, a...)}
}
