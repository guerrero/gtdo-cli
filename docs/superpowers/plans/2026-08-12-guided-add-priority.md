# Guided Add Priority Phase Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a priority step to `gtdo add -g` (typed input, empty to skip, `--only priority` support) and reorder guided phases to task → priority → contexts → projects → metadata.

**Architecture:** The priority becomes a fourth `guidedPhase` token plus a new `PromptPriority` method on the existing `addInput` boundary (TTY readline prompt and deterministic pipe line). `runGuided` runs the phases in the new order and skips the priority phase when the base task already carries a priority; `composeGuidedTask` prepends the uppercased `(A) ` prefix into the assembled text. The store path (`Store.Add`, `prepareAdd`, `priority_on_add` config) is untouched.

**Tech Stack:** Go, Cobra, `github.com/chzyer/readline`, go-internal testscript (txtar).

**Spec:** `docs/superpowers/specs/2026-08-12-guided-add-priority-design.md`

## Global Constraints

- Phase order is exactly: task, priority, contexts, projects, metadata — in both the TTY flow and the pipe protocol.
- Priority input accepts exactly two forms: an empty line (skip) or a single ASCII letter `a`–`z`/`A`–`Z`. Anything else errors with `invalid priority %q: expected a single letter A-Z` and cancels the add without writing.
- When the base task text carries a priority (detected via `todo.Task.Priority()`), the priority phase is skipped entirely: no prompt, no pipe line consumed.
- TTY prompt text is exactly `Priority (A-Z, empty to skip): `.
- Pipe protocol line order: task line, priority line, context line, project line, metadata `key:value` lines until an empty line.
- Usage string: `usage: gtdo add [-i|--interactive|-g|--guided] [--only priority|context|project|metadata]`.
- Help long text lines (pinned in help.txtar/t2100-help.txtar):
  - `Interactive modes: -i|--interactive adds one editable task; -g|--guided runs priority, context, project, and metadata phases.`
  - `Guided mode accepts repeatable --only priority|context|project|metadata.`
- `priority` is on by default (empty `Only` map enables all four phases) and is a valid `--only` value.
- No store changes: `add.go` keeps passing `cfg.PriorityOnAdd`; a composed priority suppresses the config via `prepareAdd`'s existing `priorityOnAddRe`.
- Conventional Commits; subject imperative, lower case, no trailing period, ≤72 chars. Scope `cli`.

---

### Task 1: `priority` phase token and phase-listing strings

Add the `priority` phase token and update every string that lists the phases (usage, help). Tests first.

**Files:**
- Modify: `internal/cli/add_options.go` (const block, `parseGuidedPhase`, `isGuidedPhase`, `addUsage`)
- Modify: `internal/cli/actions.go:38` (add long help)
- Test: `internal/cli/add_options_test.go`
- Test: `internal/cli/testdata/script/t2300-interactive-add.txtar` (usage line only)
- Test: `internal/cli/testdata/script/help.txtar` (lines 75–76)
- Test: `internal/cli/testdata/script/t2100-help.txtar` (lines 87–88)

**Interfaces:**
- Consumes: existing `guidedPhase`, `parseGuidedPhase`, `isGuidedPhase`, `addUsage`.
- Produces: `phasePriority guidedPhase = "priority"` (declared first in the const block); `parseGuidedPhase("priority")` returns it without error; `isGuidedPhase(phasePriority)` returns true; `addUsage()` lists `priority|context|project|metadata`.

- [ ] **Step 1: Write the failing tests**

In `internal/cli/add_options_test.go`:

1. Add a parse case to `TestParseAddOptions`:
```go
{name: "guided priority only", args: []string{"-g", "--only", "priority"}, mode: addModeGuided, only: []guidedPhase{phasePriority}},
```
2. Extend the phase-flag loop in the same test so `phasePriority` is checked too:
```go
for _, phase := range []guidedPhase{phasePriority, phaseMetadata, phaseProject, phaseContext} {
```
3. Add a case to `TestAddOptionsPhaseEnabled`:
```go
{name: "empty selection enables priority", phase: phasePriority, want: true},
```
4. Extend `TestAddUsage`'s contains list:
```go
for _, name := range []string{"-i", "--interactive", "-g", "--guided", "--only", "priority|context|project|metadata"} {
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestParseAddOptions|TestAddOptionsPhaseEnabled|TestAddUsage'`
Expected: FAIL — `parseGuidedPhase("priority")` returns `unknown guided phase "priority"`; usage string lacks `priority|context|project|metadata`.

- [ ] **Step 3: Add the phase token and update the listing strings**

In `internal/cli/add_options.go`:

1. Reorder and extend the phase constants to the canonical phase order:
```go
const (
	phasePriority guidedPhase = "priority"
	phaseContext  guidedPhase = "context"
	phaseProject  guidedPhase = "project"
	phaseMetadata guidedPhase = "metadata"
)
```
2. Extend `parseGuidedPhase`:
```go
	switch phase {
	case phasePriority, phaseContext, phaseProject, phaseMetadata:
		return phase, nil
```
3. Extend `isGuidedPhase`:
```go
	switch phase {
	case phasePriority, phaseContext, phaseProject, phaseMetadata:
		return true
```
4. Update `addUsage`:
```go
func addUsage() string {
	return fmt.Sprintf("usage: %s add [-i|--interactive|-g|--guided] [--only priority|context|project|metadata]", ProgName)
}
```

In `internal/cli/actions.go:38`, replace the long help text:
```go
		long:    "Adds THING I NEED TO DO to your todo.txt file on its own line.\nProject and context notation optional.\nQuotes optional.\nInteractive modes: -i|--interactive adds one editable task; -g|--guided runs priority, context, project, and metadata phases.\nGuided mode accepts repeatable --only priority|context|project|metadata.",
```

- [ ] **Step 4: Update the pinned txtar help/usage lines**

In `internal/cli/testdata/script/help.txtar`, replace lines 75–76 with:
```text
      Interactive modes: -i|--interactive adds one editable task; -g|--guided runs priority, context, project, and metadata phases.
      Guided mode accepts repeatable --only priority|context|project|metadata.
```
Apply the same two-line replacement to `internal/cli/testdata/script/t2100-help.txtar` (lines 87–88).

In `internal/cli/testdata/script/t2300-interactive-add.txtar`, replace only the usage line in the `invalid.stderr.want` section:
```text
usage: gtdo add [-i|--interactive|-g|--guided] [--only priority|context|project|metadata]
```
Do not touch the guided scenarios yet — the line protocol is still the old one until Task 3.

- [ ] **Step 5: Run the full suite**

Run: `go test ./...`
Expected: PASS (unit tests + all txtar scripts).

- [ ] **Step 6: Commit**

```bash
git add internal/cli/add_options.go internal/cli/actions.go internal/cli/add_options_test.go internal/cli/testdata/script/help.txtar internal/cli/testdata/script/t2100-help.txtar internal/cli/testdata/script/t2300-interactive-add.txtar
git commit -m "feat(cli): add priority guided phase token"
```

---

### Task 2: Priority input boundary (`PromptPriority` on both adapters)

Add `PromptPriority` to the `addInput` interface and implement it for the line and TTY adapters, with the shared validator. Tests first.

**Files:**
- Modify: `internal/cli/add_guided.go` (interface, `lineAddInput.PromptPriority`, `parseGuidedPriority`, `isASCIILetter`)
- Modify: `internal/cli/add_terminal.go` (`ttyAddInput.PromptPriority`, inserted right after `PromptTask`)
- Test: `internal/cli/add_guided_test.go` (fake gains the method + new adapter tests)

**Interfaces:**
- Consumes: existing `addInput`, `lineAddInput`, `ttyAddInput`, `readline`, `readLineErr`.
- Produces:
  - `addInput.PromptPriority(addCandidates) (string, error)` — returns `""` for an empty line or the single letter as typed; error for anything else.
  - `lineAddInput.PromptPriority(addCandidates) (string, error)` — consumes exactly one line.
  - `ttyAddInput.PromptPriority(addCandidates) (string, error)` — readline prompt `Priority (A-Z, empty to skip): `.
  - `parseGuidedPriority(line string) (string, error)` — validates; returns the letter unchanged (lowercase stays lowercase; `composeGuidedTask` uppercases in Task 3) or `""`.
  - `isASCIILetter(char byte) bool` — `a`–`z` or `A`–`Z`.

- [ ] **Step 1: Write the failing tests**

In `internal/cli/add_guided_test.go`:

1. Add the `priority` field and method to `fakeAddInput` (needed for compilation; the field feeds Task 3's flow tests):
```go
type fakeAddInput struct {
	task       string
	priority   string
	metadata   []string
	selections map[guidedPhase][]string
	calls      []guidedPhase
	selectArgs map[guidedPhase][]string
}

func (f *fakeAddInput) PromptPriority(addCandidates) (string, error) {
	f.calls = append(f.calls, phasePriority)
	return f.priority, nil
}
```
2. Add adapter tests:
```go
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
			}
		})
	}
}

func TestLineAddInputPropagatesPriorityEOF(t *testing.T) {
	input := lineAddInput{reader: bufio.NewReader(strings.NewReader("Call team"))}
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
```
3. Add a direct validator test:
```go
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
		if _, err := parseGuidedPriority(line); err == nil {
			t.Fatalf("parseGuidedPriority(%q) succeeded, want error", line)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestLineAddInput|TestParseGuidedPriority'`
Expected: FAIL — package does not compile: `PromptPriority` not in the `addInput` interface, no `parseGuidedPriority` function.

- [ ] **Step 3: Implement the interface method and validator**

In `internal/cli/add_guided.go`:

1. Extend the interface:
```go
type addInput interface {
	PromptTask(addCandidates) (string, error)
	PromptPriority(addCandidates) (string, error)
	PromptMetadata(addCandidates) ([]string, error)
	Select(guidedPhase, []string) ([]string, error)
}
```
2. Add the line adapter method after `PromptTask`:
```go
// PromptPriority consumes one priority line. The shared validator accepts
// an empty line (skip) or a single ASCII letter.
func (l lineAddInput) PromptPriority(addCandidates) (string, error) {
	line, err := l.readLineErr()
	if err != nil {
		return "", err
	}
	return parseGuidedPriority(line)
}
```
3. Add the validator next to `validateMetadataLine`:
```go
// parseGuidedPriority validates a guided priority line: empty, or a single
// ASCII letter. The letter is returned as typed; composition uppercases it.
func parseGuidedPriority(line string) (string, error) {
	if line == "" {
		return "", nil
	}
	if len(line) != 1 || !isASCIILetter(line[0]) {
		return "", fmt.Errorf("invalid priority %q: expected a single letter A-Z", line)
	}
	return line, nil
}

func isASCIILetter(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}
```

In `internal/cli/add_terminal.go`, add the TTY adapter method right after `PromptTask`:
```go
// PromptPriority asks for a single priority letter. An empty line skips the
// phase; the shared validator rejects anything that is not one letter. No
// completion is offered: the letter set is tiny and typed directly.
func (t *ttyAddInput) PromptPriority(addCandidates) (string, error) {
	line, err := t.readline("Priority (A-Z, empty to skip): ", stringCompleter{})
	if err != nil {
		return "", err
	}
	return parseGuidedPriority(line)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/cli/`
Expected: PASS — all adapter and validator tests pass; existing tests unchanged (the new methods are not yet called by `runGuided`).

- [ ] **Step 5: Commit**

```bash
git add internal/cli/add_guided.go internal/cli/add_terminal.go internal/cli/add_guided_test.go
git commit -m "feat(cli): prompt guided priority input"
```

---

### Task 3: Guided flow — new phase order, skip rule, composition

Rewire `runGuided` to the new phase order with the priority skip rule, extend `composeGuidedTask`, update flow tests, and update the scripted guided scenarios in t2300. Tests first.

**Files:**
- Modify: `internal/cli/add_guided.go` (`runGuided`, `composeGuidedTask`, add `prependGuidedToken`)
- Test: `internal/cli/add_guided_test.go` (flow/composition tests, protocol tests)
- Test: `internal/cli/testdata/script/t2300-interactive-add.txtar` (protocol, outputs, new scenarios)
- Docs: `CHANGELOG.md` (Unreleased → Added)

**Interfaces:**
- Consumes: `PromptPriority` (Task 2), `phasePriority` (Task 1), `todo.Task.Priority()`.
- Produces:
  - `runGuided(input addInput, candidates addCandidates, opts addOptions) (string, error)` — task, then priority (only when the base has no priority), then contexts, projects, metadata.
  - `composeGuidedTask(base, priority string, contexts, projects, metadata []string) string` — prepends `"("+strings.ToUpper(priority)+")"` as the `(A) ` prefix when non-empty, then appends context, project, metadata groups with the existing dedup/sort rules.
  - `prependGuidedToken(text, token string) string` — joins token before text with one space; empty either side returns the other.

- [ ] **Step 1: Write the failing tests**

In `internal/cli/add_guided_test.go`:

1. Replace `TestRunGuidedRunsSelectedPhasesInOrder` with the new order and priority:
```go
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
```
2. Add the skip-rule tests:
```go
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
```
3. Replace `TestComposeGuidedTaskSkipsDuplicateSigilsAndEmptyGroups` with the new signature and priority cases:
```go
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
			want:     "(A) due:tomorrow +gtdo",
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
```
4. Update the two protocol tests for the new line order:

`TestLineAddInputReadsTaskAndSelections` — new input stream (task, priority, contexts, projects, metadata) and a priority assertion:
```go
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
```
`TestLineAddInputTreatsEmptyLinesAsEmptySelections` — one more empty line (priority) in the input and an assertion:
```go
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
```
`TestRunGuidedRunsOnlySelectedPhaseAfterTask` keeps its body unchanged — the fake now compiles with the `priority` field unused.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestRunGuided|TestComposeGuidedTask|TestLineAddInput'`
Expected: FAIL — `runGuided` still runs the old order, `composeGuidedTask` has the old signature (compile error).

- [ ] **Step 3: Implement the flow changes**

In `internal/cli/add_guided.go`:

1. Replace `runGuided`:
```go
// runGuided gathers the base task and each enabled optional phase in the
// stable priority, context, project, metadata order before composing one
// final line. The priority phase is skipped when the base task already
// carries a priority, mirroring the duplicate-sigil rule.
func runGuided(input addInput, candidates addCandidates, opts addOptions) (string, error) {
	base, err := input.PromptTask(candidates)
	if err != nil {
		return "", err
	}

	var priority string
	if opts.phaseEnabled(phasePriority) {
		if _, has := (todo.Task{Text: base}).Priority(); !has {
			priority, err = input.PromptPriority(candidates)
			if err != nil {
				return "", err
			}
		}
	}

	var contexts, projects, metadata []string
	if opts.phaseEnabled(phaseContext) {
		contexts, err = input.Select(phaseContext, candidates.Contexts)
		if err != nil {
			return "", err
		}
	}
	if opts.phaseEnabled(phaseProject) {
		projects, err = input.Select(phaseProject, candidates.Projects)
		if err != nil {
			return "", err
		}
	}
	if opts.phaseEnabled(phaseMetadata) {
		metadata, err = input.PromptMetadata(candidates)
		if err != nil {
			return "", err
		}
	}

	return composeGuidedTask(base, priority, contexts, projects, metadata), nil
}
```
2. Replace `composeGuidedTask` and add the prepend helper:
```go
// composeGuidedTask prepends a non-empty priority, then appends non-empty
// context, project, and metadata groups with one separator at each boundary.
// Existing exact sigil words in base are retained and are not appended a
// second time; runGuided guarantees the base carries no priority when one
// is passed here.
func composeGuidedTask(base, priority string, contexts, projects, metadata []string) string {
	text := base
	if priority != "" {
		text = prependGuidedToken(text, "("+strings.ToUpper(priority)+")")
	}

	contextSet := make(map[string]struct{})
	for _, context := range (todo.Task{Text: base}).Contexts() {
		contextSet[context] = struct{}{}
	}
	for _, context := range sortedGuidedTokens(contexts) {
		if context == "" {
			continue
		}
		if _, exists := contextSet[context]; exists {
			continue
		}
		contextSet[context] = struct{}{}
		text = appendGuidedToken(text, context)
	}

	projectSet := make(map[string]struct{})
	for _, project := range (todo.Task{Text: base}).Projects() {
		projectSet[project] = struct{}{}
	}
	for _, project := range sortedGuidedTokens(projects) {
		if project == "" {
			continue
		}
		if _, exists := projectSet[project]; exists {
			continue
		}
		projectSet[project] = struct{}{}
		text = appendGuidedToken(text, project)
	}

	for _, pair := range metadata {
		text = appendGuidedToken(text, pair)
	}

	return text
}

// prependGuidedToken joins token before text with one separator; an empty
// either side returns the other side unchanged.
func prependGuidedToken(text, token string) string {
	if token == "" {
		return text
	}
	if text == "" {
		return token
	}
	return token + " " + text
}
```
The `sortedGuidedTokens` and `appendGuidedToken` helpers are unchanged.

- [ ] **Step 4: Update the t2300 scripted scenarios**

Replace the whole `internal/cli/testdata/script/t2300-interactive-add.txtar` with:
```text
# candidates are a union of todo.txt and done.txt; report.txt is a decoy
stdin interactive.in
exec gtdo add -i
cmp stdout interactive.want
cmp stderr empty.want
cmp todo.txt interactive.todo.want
cmp done.txt done.want

stdin guided.in
exec gtdo add -g
cmp stdout guided.want
cmp stderr empty.want
cmp todo.txt guided.todo.want
cmp done.txt done.want

stdin context-only.in
exec gtdo add -g --only context
cmp stdout context-only.want
cmp stderr empty.want
cmp todo.txt context-only.todo.want
cmp done.txt done.want

stdin priority-only.in
exec gtdo add -g --only priority
cmp stdout priority-only.want
cmp stderr empty.want
cmp todo.txt priority-only.todo.want
cmp done.txt done.want

stdin existing-priority.in
exec gtdo add -g
cmp stdout existing-priority.want
cmp stderr empty.want
cmp todo.txt existing-priority.todo.want
cmp done.txt done.want

! exec status1 gtdo add -i extra text
cmp stdout empty.want
cmp stderr invalid.stderr.want
cmp todo.txt existing-priority.todo.want
cmp done.txt done.want

-- todo.txt --
current @home +gtdo due:today
-- done.txt --
old @phone +legacy status:done
-- report.txt --
report @ignored +ignored
-- interactive.in --
follow up @phone +legacy
-- guided.in --
review current @home
A
@home @phone @ignored
+gtdo +legacy
due:tomorrow

-- context-only.in --
quick check @home

@home @phone @ignored
-- priority-only.in --
ship the release
b
-- existing-priority.in --
(B) urgent fix @home
@phone
+gtdo
due:tomorrow

-- interactive.want --
2 follow up @phone +legacy
TODO: 2 added.
-- interactive.todo.want --
current @home +gtdo due:today
follow up @phone +legacy
-- guided.want --
3 (A) review current @home @phone +gtdo +legacy due:tomorrow
TODO: 3 added.
-- guided.todo.want --
current @home +gtdo due:today
follow up @phone +legacy
(A) review current @home @phone +gtdo +legacy due:tomorrow
-- context-only.want --
4 quick check @home @phone
TODO: 4 added.
-- context-only.todo.want --
current @home +gtdo due:today
follow up @phone +legacy
(A) review current @home @phone +gtdo +legacy due:tomorrow
quick check @home @phone
-- priority-only.want --
5 (B) ship the release
TODO: 5 added.
-- priority-only.todo.want --
current @home +gtdo due:today
follow up @phone +legacy
(A) review current @home @phone +gtdo +legacy due:tomorrow
quick check @home @phone
(B) ship the release
-- existing-priority.want --
6 (B) urgent fix @home @phone +gtdo due:tomorrow
TODO: 6 added.
-- existing-priority.todo.want --
current @home +gtdo due:today
follow up @phone +legacy
(A) review current @home @phone +gtdo +legacy due:tomorrow
quick check @home @phone
(B) ship the release
(B) urgent fix @home @phone +gtdo due:tomorrow
-- done.want --
old @phone +legacy status:done
-- invalid.stderr.want --
usage: gtdo add [-i|--interactive|-g|--guided] [--only priority|context|project|metadata]
-- empty.want --
```
Notes:
- `guided.in` has no priority line because... it does: line 2 is `A` (empty-priority skip is covered by `context-only.in`'s blank line). The trailing blank line after `due:tomorrow` terminates the metadata phase.
- `existing-priority.in` has **no** priority line: the base carries `(B)`, so the phase is skipped and no line is consumed — the context line follows the task line directly.
- `@ignored` is filtered out because it is not a candidate (report.txt is never scanned).

- [ ] **Step 5: Add the changelog entry**

In `CHANGELOG.md`, under `## [Unreleased]` → `### Added`, after the `enableUUID` bullet, add:
```markdown
- Guided `add -g` now asks for an optional priority (`--only priority`) and
  runs its phases in task, priority, context, project, metadata order.
```

- [ ] **Step 6: Run the full suite**

Run: `go test ./...`
Expected: PASS — flow tests, protocol tests, and all txtar scripts.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/add_guided.go internal/cli/add_guided_test.go internal/cli/testdata/script/t2300-interactive-add.txtar CHANGELOG.md
git commit -m "feat(cli): compose guided priority phase"
```

---

### Task 4: Full verification

Confirm the whole repository is green and the spec is fully covered.

**Files:**
- None expected to change; fix anything the checks surface.

- [ ] **Step 1: Run the full gate**

Run: `make test && make lint && make build`
Expected: all pass; gofumpt reports no formatting diffs.

- [ ] **Step 2: Cross-check the spec**

Walk `docs/superpowers/specs/2026-08-12-guided-add-priority-design.md` section by section against the code:
- Command contract: `--only priority` accepted, default-on, usage/help strings — Task 1.
- Skip rule: `runGuided` skips when `todo.Task.Priority()` matches; no pipe line consumed — Task 3.
- Input/validation: `parseGuidedPriority` with the exact error text; TTY prompt string — Task 2.
- Pipe protocol order: task, priority, context, project, metadata — Tasks 2–3.
- Composition: `(A) ` prefix, uppercasing, empty composes nothing — Task 3.
- Store untouched: `add.go` still passes `cfg.PriorityOnAdd` — confirm with `git diff main..HEAD -- internal/cli/add.go` showing no changes.
- Testing: unit + txtar coverage — Tasks 1–3.

Fix any gap found, then re-run `make test && make lint && make build`.

- [ ] **Step 3: Commit any verification fixes**

If Step 2 changed files, commit with a `fix(cli):` or `test(cli):` message. If nothing changed, there is no commit for this task.
