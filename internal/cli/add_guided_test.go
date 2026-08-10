package cli

import (
	"bufio"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type fakeAddInput struct {
	task       string
	metadata   []string
	selections map[guidedPhase][]string
	calls      []guidedPhase
	selectArgs map[guidedPhase][]string
}

func (f *fakeAddInput) PromptTask(addCandidates) (string, error) {
	return f.task, nil
}

func (f *fakeAddInput) PromptMetadata(addCandidates) ([]string, error) {
	f.calls = append(f.calls, phaseMetadata)
	return f.metadata, nil
}

func (f *fakeAddInput) Select(phase guidedPhase, options []string) ([]string, error) {
	f.calls = append(f.calls, phase)
	if f.selectArgs == nil {
		f.selectArgs = make(map[guidedPhase][]string)
	}
	f.selectArgs[phase] = append([]string(nil), options...)
	return f.selections[phase], nil
}

func TestRunGuidedRunsSelectedPhasesInOrder(t *testing.T) {
	input := &fakeAddInput{
		task:     "Call team @home",
		metadata: []string{"due:tomorrow"},
		selections: map[guidedPhase][]string{
			phaseProject: {"+gtdo"},
			phaseContext: {"@phone", "@home"},
		},
	}

	got, err := runGuided(input, addCandidates{}, addOptions{Mode: addModeGuided})
	if err != nil {
		t.Fatal(err)
	}
	want := "Call team @home due:tomorrow +gtdo @phone"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !reflect.DeepEqual(input.calls, []guidedPhase{phaseMetadata, phaseProject, phaseContext}) {
		t.Fatalf("calls = %v, want metadata, project, context", input.calls)
	}
}

func TestRunGuidedRunsOnlySelectedPhaseAfterTask(t *testing.T) {
	input := &fakeAddInput{
		task:     "Call team",
		metadata: []string{"due:tomorrow"},
		selections: map[guidedPhase][]string{
			phaseContext: {"@phone"},
		},
	}

	got, err := runGuided(input, addCandidates{}, addOptions{
		Mode: addModeGuided,
		Only: map[guidedPhase]bool{phaseContext: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Call team @phone" {
		t.Fatalf("got %q, want %q", got, "Call team @phone")
	}
	if !reflect.DeepEqual(input.calls, []guidedPhase{phaseContext}) {
		t.Fatalf("calls = %v, want context only", input.calls)
	}
}

func TestComposeGuidedTaskSkipsDuplicateSigilsAndEmptyGroups(t *testing.T) {
	tests := []struct {
		name                         string
		base                         string
		metadata, projects, contexts []string
		want                         string
	}{
		{
			name:     "groups separated by one space",
			base:     "Call team",
			metadata: []string{"due:tomorrow", "status:open"},
			projects: []string{"+gtdo"},
			contexts: []string{"@home"},
			want:     "Call team due:tomorrow status:open +gtdo @home",
		},
		{
			name:     "empty base and groups",
			metadata: []string{"due:tomorrow"},
			projects: []string{"+gtdo"},
			want:     "due:tomorrow +gtdo",
		},
		{
			name:     "duplicate sigils",
			base:     "Call team +gtdo @home",
			projects: []string{"+gtdo", "+other"},
			contexts: []string{"@home", "@phone"},
			want:     "Call team +gtdo @home +other @phone",
		},
		{
			name:     "empty selections",
			base:     "Call team",
			metadata: []string{},
			projects: []string{},
			contexts: []string{},
			want:     "Call team",
		},
		{
			name:     "sigils sorted before composition",
			base:     "Call team",
			projects: []string{"+z", "+a"},
			contexts: []string{"@z", "@a"},
			want:     "Call team +a +z @a @z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := composeGuidedTask(tc.base, tc.metadata, tc.projects, tc.contexts); got != tc.want {
				t.Fatalf("composeGuidedTask() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLineAddInputReadsTaskAndSelections(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\ndue:tomorrow\nstatus:open\n\n+gtdo +missing\n@home @missing\n"))}
	candidates := addCandidates{Projects: []string{"+gtdo"}, Contexts: []string{"@home"}}

	task, err := input.PromptTask(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if task != "Call team" {
		t.Fatalf("task = %q, want Call team", task)
	}
	metadata, err := input.PromptMetadata(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(metadata, []string{"due:tomorrow", "status:open"}) {
		t.Fatalf("metadata = %v, want due/status", metadata)
	}
	projects, err := input.Select(phaseProject, candidates.Projects)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projects, []string{"+gtdo"}) {
		t.Fatalf("projects = %v, want [+gtdo]", projects)
	}
	contexts, err := input.Select(phaseContext, candidates.Contexts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(contexts, []string{"@home"}) {
		t.Fatalf("contexts = %v, want [@home]", contexts)
	}
}

func TestLineAddInputTreatsEmptyLinesAsEmptySelections(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\n\n\n\n"))}
	candidates := addCandidates{}

	if _, err := input.PromptTask(candidates); err != nil {
		t.Fatal(err)
	}
	metadata, err := input.PromptMetadata(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata) != 0 {
		t.Fatalf("metadata = %v, want empty", metadata)
	}
	for _, phase := range []guidedPhase{phaseProject, phaseContext} {
		selected, err := input.Select(phase, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(selected) != 0 {
			t.Fatalf("%s selection = %v, want empty", phase, selected)
		}
	}
}

func TestLineAddInputPropagatesTaskEOF(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team"))}
	got, err := input.PromptTask(addCandidates{})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("PromptTask() error = %v, want io.EOF", err)
	}
	if got != "" {
		t.Fatalf("PromptTask() text = %q, want empty on EOF", got)
	}
}

func TestLineAddInputPropagatesSelectionEOF(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("+gtdo"))}
	got, err := input.Select(phaseProject, []string{"+gtdo"})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Select() error = %v, want io.EOF", err)
	}
	if got != nil {
		t.Fatalf("Select() values = %v, want nil on EOF", got)
	}
}

func TestLineAddInputSortsFilteredSelections(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("+z +missing +a +z\n"))}
	got, err := input.Select(phaseProject, []string{"+z", "+a"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"+a", "+z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Select() values = %v, want %v", got, want)
	}
}

func TestLineAddInputRejectsMalformedMetadataLine(t *testing.T) {
	for _, line := range []string{"missing-colon", ":value", "key:", "bad-key:value", "key value"} {
		t.Run(line, func(t *testing.T) {
			input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\n" + line + "\n"))}
			if _, err := input.PromptTask(addCandidates{}); err != nil {
				t.Fatal(err)
			}
			if _, err := input.PromptMetadata(addCandidates{}); err == nil {
				t.Fatalf("PromptMetadata(%q) succeeded, want malformed-line error", line)
			}
		})
	}
}

func TestSessionReaderUsesOneBufferedReader(t *testing.T) {
	s := &session{in: strings.NewReader("one\ntwo\nthree\n")}
	s.reader = bufio.NewReader(s.in)
	for _, want := range []string{"one", "two", "three"} {
		if got := s.readLine(); got != want {
			t.Fatalf("readLine() = %q, want %q", got, want)
		}
	}
}
