package ui

import (
	"strings"

	"github.com/guerrero/gtdo/internal/config"
)

// Color is a resolved snapshot of todo.cfg's color assignments (§5.2
// [colors]): the priority colors pri_a..pri_c with the pri_x fallback,
// color_done, the per-word colors, and default (the reset emitted after
// every colored word). Each field holds the final ANSI escape sequence;
// in plain mode every field is empty, so a zero Color renders uncolored.
// The codes are resolved by internal/config — the single source of truth
// for the 16-name color map — and FromConfig copies out the roles the
// listing pipeline asks for.
//
// Color implements todo.Colorer (§6.2.4): Color returns the field for a
// palette role ("" for anything else) and PriorityColor maps A/B/C to their
// fields with every other letter falling back to PriX, exactly todo.cfg's
// default priority map. config.Config also satisfies todo.Colorer with
// per-letter pri_a..pri_z resolution, for users who color letters beyond C.
type Color struct {
	PriA, PriB, PriC, PriX                     string
	Done, Project, Context, Date, Number, Meta string
	Default                                    string
}

// FromConfig snapshots the palette roles out of a resolved configuration.
// Config does the name→code resolution and the plain-mode short-circuit
// (§5.2): in plain mode every role resolves to "", so the resulting Color
// is plain.
func FromConfig(cfg config.Config) Color {
	return Color{
		PriA:    cfg.Color("pri_a"),
		PriB:    cfg.Color("pri_b"),
		PriC:    cfg.Color("pri_c"),
		PriX:    cfg.Color("pri_x"),
		Done:    cfg.Color("color_done"),
		Project: cfg.Color("color_project"),
		Context: cfg.Color("color_context"),
		Date:    cfg.Color("color_date"),
		Number:  cfg.Color("color_number"),
		Meta:    cfg.Color("color_meta"),
		Default: cfg.Color("default"),
	}
}

// Color returns the ANSI code for a role the pipeline asks for (§6.2.4):
// the ten palette roles, matched case-insensitively like config. Roles
// outside the palette are off, matching config.Color's behavior.
func (p Color) Color(role string) string {
	switch strings.ToLower(role) {
	case "pri_a":
		return p.PriA
	case "pri_b":
		return p.PriB
	case "pri_c":
		return p.PriC
	case "pri_x":
		return p.PriX
	case "color_done":
		return p.Done
	case "color_project":
		return p.Project
	case "color_context":
		return p.Context
	case "color_date":
		return p.Date
	case "color_number":
		return p.Number
	case "color_meta":
		return p.Meta
	case "default":
		return p.Default
	}
	return ""
}

// PriorityColor returns the color for a priority letter: A/B/C from their
// fields and every other letter — including ones a config colors
// individually — from PriX, todo.cfg's default priority map (t1330).
func (p Color) PriorityColor(letter byte) string {
	switch letter {
	case 'A':
		return p.PriA
	case 'B':
		return p.PriB
	case 'C':
		return p.PriC
	}
	return p.PriX
}
