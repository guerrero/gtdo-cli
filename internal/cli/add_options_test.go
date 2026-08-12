package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAddOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		mode    addMode
		only    []guidedPhase
		text    []string
		wantErr bool
	}{
		{name: "legacy text", args: []string{"buy", "milk"}, text: []string{"buy", "milk"}},
		{name: "legacy option-looking text", args: []string{"-task"}, text: []string{"-task"}},
		{name: "interactive shorthand", args: []string{"-i"}, mode: addModeInteractive},
		{name: "interactive long", args: []string{"--interactive"}, mode: addModeInteractive},
		{name: "guided shorthand", args: []string{"-g"}, mode: addModeGuided},
		{name: "guided priority only", args: []string{"-g", "--only", "priority"}, mode: addModeGuided, only: []guidedPhase{phasePriority}},
		{name: "guided long repeatable phases", args: []string{"--guided", "--only", "project", "--only=context"}, mode: addModeGuided, only: []guidedPhase{phaseProject, phaseContext}},
		{name: "mode with text", args: []string{"-g", "task"}, wantErr: true},
		{name: "only without guided", args: []string{"--only", "context"}, wantErr: true},
		{name: "both modes", args: []string{"-i", "--guided"}, wantErr: true},
		{name: "unknown phase", args: []string{"-g", "--only", "tags"}, wantErr: true},
		{name: "missing phase value", args: []string{"--guided", "--only"}, wantErr: true},
		{name: "empty inline phase value", args: []string{"--guided", "--only="}, wantErr: true},
		{name: "interactive with only", args: []string{"-i", "--only", "context"}, wantErr: true},
		{name: "positional after inline options", args: []string{"--guided", "--only=context", "task", "more"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAddOptions(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseAddOptions(%q) succeeded: %#v", tc.args, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAddOptions(%q): %v", tc.args, err)
			}
			if got.Mode != tc.mode {
				t.Errorf("mode = %d, want %d", got.Mode, tc.mode)
			}
			if !reflect.DeepEqual(got.Positional, tc.text) {
				t.Errorf("positional = %q, want %q", got.Positional, tc.text)
			}
			for _, phase := range []guidedPhase{phasePriority, phaseMetadata, phaseProject, phaseContext} {
				want := false
				for _, selected := range tc.only {
					if selected == phase {
						want = true
						break
					}
				}
				if got.Only[phase] != want {
					t.Errorf("only[%q] = %t, want %t (all = %#v)", phase, got.Only[phase], want, got.Only)
				}
			}
		})
	}
}

func TestAddOptionsPhaseEnabled(t *testing.T) {
	tests := []struct {
		name  string
		only  map[guidedPhase]bool
		phase guidedPhase
		want  bool
	}{
		{name: "empty selection enables priority", phase: phasePriority, want: true},
		{name: "empty selection enables metadata", phase: phaseMetadata, want: true},
		{name: "empty selection enables project", phase: phaseProject, want: true},
		{name: "empty selection enables context", phase: phaseContext, want: true},
		{name: "selected phase enabled", only: map[guidedPhase]bool{phaseProject: true}, phase: phaseProject, want: true},
		{name: "unselected phase disabled", only: map[guidedPhase]bool{phaseProject: true}, phase: phaseContext, want: false},
		{name: "unknown phase disabled", only: map[guidedPhase]bool{phaseProject: true}, phase: guidedPhase("unknown"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := (addOptions{Only: tc.only}).phaseEnabled(tc.phase); got != tc.want {
				t.Errorf("phaseEnabled(%q) = %t, want %t", tc.phase, got, tc.want)
			}
		})
	}
}

func TestAddUsage(t *testing.T) {
	usage := addUsage()
	for _, name := range []string{"-i", "--interactive", "-g", "--guided", "--only", "priority|context|project|metadata"} {
		if !strings.Contains(usage, name) {
			t.Errorf("addUsage() = %q, missing %q", usage, name)
		}
	}
}
