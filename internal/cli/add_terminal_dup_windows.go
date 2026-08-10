//go:build windows

package cli

import (
	"os"
	"syscall"
)

// duplicateTTY duplicates the Windows console or file handle without
// reopening it by name. Reopening would lose inherited console/device state;
// DuplicateHandle preserves the access and type readline expects.
func duplicateTTY(file *os.File) (*os.File, error) {
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return nil, err
	}

	var duplicate syscall.Handle
	if err := syscall.DuplicateHandle(
		process,
		syscall.Handle(file.Fd()),
		process,
		&duplicate,
		0,
		false,
		syscall.DUPLICATE_SAME_ACCESS,
	); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(duplicate), file.Name()), nil
}
