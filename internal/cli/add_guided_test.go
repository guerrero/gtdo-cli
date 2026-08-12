package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

type fakeAddInput struct {
	task       string
	priority   string
	metadata   []string
	selections map[guidedPhase][]string
	calls      []guidedPhase
	selectArgs map[guidedPhase][]string
}

func (f *fakeAddInput) PromptTask(addCandidates) (string, error) {
	return f.task, nil
}

func (f *fakeAddInput) PromptPriority(addCandidates) (string, error) {
	f.calls = append(f.calls, phasePriority)
	return f.priority, nil
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
		priority: "A",
		metadata: []string{"due:tomorrow"},
		selections: map[guidedPhase][]string{
			phaseContext: {"@phone", "@home"},
			phaseProject: {"+gtdo"},
		},
	}

	got, err := runGuided(input, addCandidates{}, addOptions{Mode: addModeGuided})
	if err != nil {
		t.Fatal(err)
	}
	want := "(A) Call team @home @phone +gtdo due:tomorrow"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !reflect.DeepEqual(input.calls, []guidedPhase{phasePriority, phaseContext, phaseProject, phaseMetadata}) {
		t.Fatalf("calls = %v, want priority, context, project, metadata", input.calls)
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

func TestRunGuidedSkipsPriorityWhenBaseHasOne(t *testing.T) {
	input := &fakeAddInput{
		task:     "(B) Call team",
		priority: "A",
		metadata: []string{"due:tomorrow"},
		selections: map[guidedPhase][]string{
			phaseContext: {"@home"},
		},
	}

	got, err := runGuided(input, addCandidates{}, addOptions{Mode: addModeGuided})
	if err != nil {
		t.Fatal(err)
	}
	if got != "(B) Call team @home due:tomorrow" {
		t.Fatalf("got %q, want %q", got, "(B) Call team @home due:tomorrow")
	}
	if !reflect.DeepEqual(input.calls, []guidedPhase{phaseContext, phaseProject, phaseMetadata}) {
		t.Fatalf("calls = %v, want context, project, metadata", input.calls)
	}
}

func TestRunGuidedOnlyPrioritySkipsWhenBaseHasOne(t *testing.T) {
	input := &fakeAddInput{task: "(B) Call team", priority: "A"}

	got, err := runGuided(input, addCandidates{}, addOptions{
		Mode: addModeGuided,
		Only: map[guidedPhase]bool{phasePriority: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "(B) Call team" {
		t.Fatalf("got %q, want %q", got, "(B) Call team")
	}
	if len(input.calls) != 0 {
		t.Fatalf("calls = %v, want none", input.calls)
	}
}

// A skipped priority phase consumes no pipe line: the context line is the
// next line after the task.
func TestRunGuidedSkipsPriorityLineInProtocol(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("(B) fix @home\n@phone\n+gtdo\ndue:tomorrow\n\n"))}
	got, err := runGuided(input, addCandidates{Contexts: []string{"@phone"}, Projects: []string{"+gtdo"}}, addOptions{Mode: addModeGuided})
	if err != nil {
		t.Fatal(err)
	}
	if got != "(B) fix @home @phone +gtdo due:tomorrow" {
		t.Fatalf("got %q, want %q", got, "(B) fix @home @phone +gtdo due:tomorrow")
	}
}

func TestComposeGuidedTaskSkipsDuplicateSigilsAndEmptyGroups(t *testing.T) {
	tests := []struct {
		name                         string
		base, priority               string
		contexts, projects, metadata []string
		want                         string
	}{
		{
			name:     "groups separated by one space",
			base:     "Call team",
			priority: "A",
			contexts: []string{"@home"},
			projects: []string{"+gtdo"},
			metadata: []string{"due:tomorrow", "status:open"},
			want:     "(A) Call team @home +gtdo due:tomorrow status:open",
		},
		{
			name:     "lowercase priority uppercased",
			base:     "Call team",
			priority: "b",
			want:     "(B) Call team",
		},
		{
			name:     "empty base with priority",
			priority: "A",
			metadata: []string{"due:tomorrow"},
			projects: []string{"+gtdo"},
			want:     "(A) +gtdo due:tomorrow",
		},
		{
			name:     "duplicate sigils",
			base:     "Call team +gtdo @home",
			contexts: []string{"@home", "@phone"},
			projects: []string{"+gtdo", "+other"},
			want:     "Call team +gtdo @home @phone +other",
		},
		{
			name: "empty selections",
			base: "Call team",
			want: "Call team",
		},
		{
			name:     "sigils sorted before composition",
			base:     "Call team",
			projects: []string{"+z", "+a"},
			contexts: []string{"@z", "@a"},
			want:     "Call team @a @z +a +z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := composeGuidedTask(tc.base, tc.priority, tc.contexts, tc.projects, tc.metadata); got != tc.want {
				t.Fatalf("composeGuidedTask() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLineAddInputReadsTaskAndSelections(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\nA\n@home @missing\n+gtdo +missing\ndue:tomorrow\nstatus:open\n\n"))}
	candidates := addCandidates{Projects: []string{"+gtdo"}, Contexts: []string{"@home"}}

	task, err := input.PromptTask(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if task != "Call team" {
		t.Fatalf("task = %q, want Call team", task)
	}
	priority, err := input.PromptPriority(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if priority != "A" {
		t.Fatalf("priority = %q, want A", priority)
	}
	contexts, err := input.Select(phaseContext, candidates.Contexts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(contexts, []string{"@home"}) {
		t.Fatalf("contexts = %v, want [@home]", contexts)
	}
	projects, err := input.Select(phaseProject, candidates.Projects)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projects, []string{"+gtdo"}) {
		t.Fatalf("projects = %v, want [+gtdo]", projects)
	}
	metadata, err := input.PromptMetadata(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(metadata, []string{"due:tomorrow", "status:open"}) {
		t.Fatalf("metadata = %v, want due/status", metadata)
	}
}

func TestLineAddInputTreatsEmptyLinesAsEmptySelections(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\n\n\n\n\n"))}
	candidates := addCandidates{}

	if _, err := input.PromptTask(candidates); err != nil {
		t.Fatal(err)
	}
	priority, err := input.PromptPriority(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if priority != "" {
		t.Fatalf("priority = %q, want empty", priority)
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
	metadata, err := input.PromptMetadata(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata) != 0 {
		t.Fatalf("metadata = %v, want empty", metadata)
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

func TestLineAddInputReadsPriorityLine(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\nb\n"))}
	if _, err := input.PromptTask(addCandidates{}); err != nil {
		t.Fatal(err)
	}
	priority, err := input.PromptPriority(addCandidates{})
	if err != nil {
		t.Fatal(err)
	}
	if priority != "b" {
		t.Fatalf("priority = %q, want b", priority)
	}
}

func TestLineAddInputSkipsEmptyPriority(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\n\n"))}
	if _, err := input.PromptTask(addCandidates{}); err != nil {
		t.Fatal(err)
	}
	priority, err := input.PromptPriority(addCandidates{})
	if err != nil {
		t.Fatal(err)
	}
	if priority != "" {
		t.Fatalf("priority = %q, want empty", priority)
	}
}

func TestLineAddInputRejectsInvalidPriority(t *testing.T) {
	for _, line := range []string{"high", "(A)", "AB", "1", "é"} {
		t.Run(line, func(t *testing.T) {
			input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\n" + line + "\n"))}
			if _, err := input.PromptTask(addCandidates{}); err != nil {
				t.Fatal(err)
			}
			if _, err := input.PromptPriority(addCandidates{}); err == nil {
				t.Fatalf("PromptPriority(%q) succeeded, want invalid-priority error", line)
			} else if got := err.Error(); got != fmt.Sprintf("invalid priority %q: expected a single letter A-Z", line) {
				t.Fatalf("PromptPriority(%q) error = %q, want %q", line, got, fmt.Sprintf("invalid priority %q: expected a single letter A-Z", line))
			}
		})
	}
}

func TestLineAddInputPropagatesPriorityEOF(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team\n"))}
	if _, err := input.PromptTask(addCandidates{}); err != nil {
		t.Fatal(err)
	}
	got, err := input.PromptPriority(addCandidates{})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("PromptPriority() error = %v, want io.EOF", err)
	}
	if got != "" {
		t.Fatalf("PromptPriority() = %q, want empty on EOF", got)
	}
}

func TestParseGuidedPriority(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", ""},
		{"a", "a"},
		{"Z", "Z"},
	} {
		got, err := parseGuidedPriority(tc.in)
		if err != nil {
			t.Fatalf("parseGuidedPriority(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseGuidedPriority(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, line := range []string{"high", "(A)", "AB", "1", "é"} {
		_, err := parseGuidedPriority(line)
		if err == nil {
			t.Fatalf("parseGuidedPriority(%q) succeeded, want error", line)
		}
		if got := err.Error(); got != fmt.Sprintf("invalid priority %q: expected a single letter A-Z", line) {
			t.Fatalf("parseGuidedPriority(%q) error = %q, want %q", line, got, fmt.Sprintf("invalid priority %q: expected a single letter A-Z", line))
		}
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
