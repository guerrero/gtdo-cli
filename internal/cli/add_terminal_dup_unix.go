//go:build !windows

package cli

import (
	"os"
	"syscall"
)

// duplicateTTY keeps Unix PTY reads independent from the caller's descriptor
// so closing readline cannot consume or close the session's original stdin.
func duplicateTTY(file *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), file.Name()), nil
}
