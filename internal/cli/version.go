package cli

import (
	"fmt"
	"runtime"
	"strings"
)

// Populated at build time via -ldflags -X github.com/guerrero/gtdo/internal/cli.Version=...
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// VersionString renders gtdo's own version block (spec §6.4), newline-terminated.
func VersionString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "gtdo %s\n", Version)
	fmt.Fprintf(&b, "commit:  %s\n", Commit)
	fmt.Fprintf(&b, "built:   %s\n", Date)
	fmt.Fprintf(&b, "go:      %s\n", runtime.Version())
	return b.String()
}
