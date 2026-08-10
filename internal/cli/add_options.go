package cli

import (
	"fmt"
	"strings"
)

// addMode identifies the optional input mode selected for add. The zero value
// keeps the existing positional add path unchanged.
type addMode uint8

const (
	addModeNone addMode = iota
	addModeInteractive
	addModeGuided
)

// guidedPhase identifies one of the optional guided-input phases.
type guidedPhase string

const (
	phaseMetadata guidedPhase = "metadata"
	phaseProject  guidedPhase = "project"
	phaseContext  guidedPhase = "context"
)

// addOptions contains the action-local options consumed by add. Positional
// text is kept separate so legacy invocations can continue through addInput.
type addOptions struct {
	Mode       addMode
	Only       map[guidedPhase]bool
	Positional []string
}

// parseAddOptions parses only add's leading mode options. Once an unrecognized
// word is encountered, it and every following word are task text, preserving
// legacy arguments such as "-task".
func parseAddOptions(args []string) (addOptions, error) {
	var opts addOptions

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-i", "--interactive":
			if opts.Mode == addModeGuided {
				return addOptions{}, fmt.Errorf("interactive and guided modes are mutually exclusive")
			}
			opts.Mode = addModeInteractive
		case "-g", "--guided":
			if opts.Mode == addModeInteractive {
				return addOptions{}, fmt.Errorf("interactive and guided modes are mutually exclusive")
			}
			opts.Mode = addModeGuided
		case "--only":
			if i+1 >= len(args) {
				return addOptions{}, fmt.Errorf("--only requires a phase")
			}
			i++
			phase, err := parseGuidedPhase(args[i])
			if err != nil {
				return addOptions{}, err
			}
			if opts.Only == nil {
				opts.Only = make(map[guidedPhase]bool)
			}
			opts.Only[phase] = true
		default:
			if strings.HasPrefix(arg, "--only=") {
				phase, err := parseGuidedPhase(strings.TrimPrefix(arg, "--only="))
				if err != nil {
					return addOptions{}, err
				}
				if opts.Only == nil {
					opts.Only = make(map[guidedPhase]bool)
				}
				opts.Only[phase] = true
				continue
			}

			opts.Positional = append(opts.Positional, args[i:]...)
			i = len(args)
		}
	}

	if len(opts.Only) > 0 && opts.Mode != addModeGuided {
		return addOptions{}, fmt.Errorf("--only requires guided mode")
	}
	if opts.Mode != addModeNone && len(opts.Positional) > 0 {
		return addOptions{}, fmt.Errorf("interactive and guided modes do not accept positional text")
	}
	return opts, nil
}

func parseGuidedPhase(value string) (guidedPhase, error) {
	phase := guidedPhase(value)
	switch phase {
	case phaseMetadata, phaseProject, phaseContext:
		return phase, nil
	default:
		if value == "" {
			return "", fmt.Errorf("--only requires a phase")
		}
		return "", fmt.Errorf("unknown guided phase %q", value)
	}
}

// phaseEnabled reports whether phase should run. An empty Only map means all
// known phases are enabled; a non-empty map is an explicit phase selection.
func (o addOptions) phaseEnabled(phase guidedPhase) bool {
	if !isGuidedPhase(phase) {
		return false
	}
	if len(o.Only) == 0 {
		return true
	}
	return o.Only[phase]
}

func isGuidedPhase(phase guidedPhase) bool {
	switch phase {
	case phaseMetadata, phaseProject, phaseContext:
		return true
	default:
		return false
	}
}

// addUsage describes add's action-local mode options for usage failures.
func addUsage() string {
	return fmt.Sprintf("usage: %s add [-i|--interactive|-g|--guided] [--only metadata|project|context]", ProgName)
}
