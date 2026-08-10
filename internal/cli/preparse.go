package cli

import (
	"errors"

	"github.com/guerrero/gtdo/internal/config"
)

// errUsage signals an unknown flag or a -d without an argument; the caller
// prints the usage text to stdout and exits 1.
var errUsage = errors.New("usage")

// Preparse scans args exactly like todo.sh's getopts loop: global flags are
// accepted only before the first non-flag argument, which is the action.
// It returns the action name ("" when none) and the action's arguments.
//
// -h short-circuits: it forwards to the shorthelp action and discards every
// remaining argument, mirroring todo.sh's `set -- '-h' 'shorthelp'`.
// -V short-circuits too: it marks Version and discards the rest.
func Preparse(args []string) (*config.Options, string, []string, error) {
	opts := &config.Options{}
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			break // first non-flag: the action
		}
		for j := 1; j < len(arg); j++ {
			switch c := arg[j]; c {
			case '@':
				opts.HideContexts = !opts.HideContexts
			case '+':
				opts.HideProjects = !opts.HideProjects
			case 'a':
				opts.AutoArchive, opts.AutoArchiveSet = false, true
			case 'A':
				opts.AutoArchive, opts.AutoArchiveSet = true, true
			case 'c':
				opts.Plain, opts.PlainSet = false, true
			case 'd':
				if j+1 < len(arg) {
					opts.ConfigPath = arg[j+1:]
				} else if i+1 < len(args) {
					i++
					opts.ConfigPath = args[i]
				} else {
					return nil, "", nil, errUsage
				}
				j = len(arg) // -d consumed the rest of the cluster
			case 'f':
				opts.Force, opts.ForceSet = true, true
			case 'h':
				return opts, "shorthelp", nil, nil
			case 'n':
				opts.Preserve, opts.PreserveSet = false, true
			case 'N':
				opts.Preserve, opts.PreserveSet = true, true
			case 'p':
				opts.Plain, opts.PlainSet = true, true
			case 'P':
				opts.HidePriority = !opts.HidePriority
			case 'v':
				opts.VerboseCount++
			case 'V':
				opts.Version = true
				return opts, "", nil, nil
			case 'x':
				// Accepted as a no-op for CLI compatibility (spec §2).
			default:
				return nil, "", nil, errUsage
			}
		}
		i++
	}
	if i >= len(args) {
		return opts, "", nil, nil
	}
	return opts, args[i], args[i+1:], nil
}
