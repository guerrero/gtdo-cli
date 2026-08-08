package cli

// The priority actions (§6.3): pri and depri. Both process their task
// numbers one pair (or one item) at a time, mirroring todo.sh's loops:
// stdout carries the per-item confirmations, stderr the already/not
// prioritized complaints, and the exit status is 1 when any complaint
// was raised.

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
)

// actionPri sets or replaces priorities: `pri NR PRIORITY [NR PRIORITY
// ...]`. A lowercase PRIORITY is upper-cased first; anything that is not
// exactly one letter A-Z dies with the two-line usage, like todo.sh's
// `@([A-Z])` extglob check.
func actionPri(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	usage := `usage: ` + ProgName + ` pri NR PRIORITY [NR PRIORITY ...]
note: PRIORITY must be anywhere from A to Z.`
	status := exitcode.OK
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			// todo.sh's `[ $# -lt 2 ] && die` inside the while loop:
			// `pri 1 A 2` dies after the first pair was processed, and
			// a bare `pri` succeeds silently (loop never entered).
			return s.die(usage)
		}
		item, newpri := args[i], strings.ToUpper(args[i+1])
		if len(newpri) != 1 || newpri[0] < 'A' || newpri[0] > 'Z' {
			return s.die(usage)
		}
		n, ok := parseItem(item)
		if !ok {
			return s.die(usage)
		}
		res, err := s.store.Pri(n, newpri[0])
		if err != nil {
			return s.die(err.Error())
		}
		if s.verbose() {
			fmt.Fprintf(s.out, "%d %s\n", n, res.NewText)
			if res.OldPri != res.NewPri {
				if res.OldPri != 0 {
					fmt.Fprintf(s.out, "TODO: %s re-prioritized from (%c) to (%c).\n", item, res.OldPri, res.NewPri)
				} else {
					fmt.Fprintf(s.out, "TODO: %s prioritized (%c).\n", item, res.NewPri)
				}
			}
		}
		if res.OldPri == res.NewPri {
			fmt.Fprintf(s.errw, "TODO: %s already prioritized (%c).\n", item, res.NewPri)
			status = exitcode.Generic
		}
	}
	if status != exitcode.OK {
		return exitcode.Wrap(status, exitcode.ErrFailure)
	}
	return nil
}

// actionDepri removes priorities: `depri NR [NR ...]`, with commas
// splitting like spaces (todo.sh's `${*//,/ }`). Items are processed one
// at a time like todo.sh's loop: a bad item dies after the earlier ones
// have already been handled, and an unprioritized item is a stderr
// complaint that leaves the exit status at 1.
func actionDepri(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	usage := `usage: ` + ProgName + ` depri NR [NR ...]`
	if len(args) == 0 {
		return s.die(usage)
	}
	status := exitcode.OK
	for _, raw := range splitItems(args) {
		n, ok := parseItem(raw)
		if !ok {
			return s.die(usage)
		}
		results, err := s.store.Depri([]int{n})
		for _, res := range results {
			if res.Prioritized {
				if s.verbose() {
					fmt.Fprintf(s.out, "%d %s\n", res.LineNumber, res.NewText)
					fmt.Fprintf(s.out, "TODO: %d deprioritized.\n", res.LineNumber)
				}
			} else {
				fmt.Fprintf(s.errw, "TODO: %d is not prioritized.\n", res.LineNumber)
				status = exitcode.Generic
			}
		}
		if err != nil {
			return s.die(err.Error())
		}
	}
	if status != exitcode.OK {
		return exitcode.Wrap(status, exitcode.ErrFailure)
	}
	return nil
}
