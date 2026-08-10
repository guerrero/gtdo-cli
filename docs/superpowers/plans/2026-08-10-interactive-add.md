# Interactive Add Modes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task with verification checkpoints. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gtdo add -i/--interactive` with inline `@`/`+` completion and `gtdo add -g/--guided` with metadata, project, and context phases, while preserving the existing add contract.

**Architecture:** An action-local parser separates the new modes from legacy positional adds. A pure candidate collector reads only configured `todo_file` and `done_file` paths and returns sorted sigils plus metadata values. An input boundary provides a `github.com/chzyer/readline` TTY implementation (including a raw selector loop) and a deterministic line-oriented implementation for pipes/tests; the final assembled text is sent once through the existing `Store.Add` path.

**Tech Stack:** Go 1.26.5, Cobra, `github.com/chzyer/readline` v1.5.1, existing `internal/config`, `internal/todo`, and go-internal testscript.

## Global Constraints

- `-i` is shorthand for `--interactive`; `-g` is shorthand for `--guided`.
- `--only` is repeatable, requires guided mode, and accepts exactly `metadata`, `project`, or `context`.
- Existing `gtdo add "text"` and `gtdo add` behavior, prompts, output, exit codes, and file bytes remain unchanged.
- Candidate data comes only from existing configured `todo_file` and `done_file`; `report_file` is never scanned.
- Candidate reads are best-effort and the collector itself never creates files.
- Sigil and metadata candidates are deduplicated and sorted byte-wise.
- TTY input has editable prompts and dynamic `@`/`+` completion; non-TTY input has no escape sequences and follows a documented line protocol.
- Guided phases run in metadata, project, context order; an empty selection is valid and task text is always requested.
- The final task is passed once to the existing `Store.Add` pipeline, retaining date-on-add, priority-on-add, validation, numbering, and confirmation semantics.
- Invalid modes, unknown phases, unsupported mode positional arguments, EOF, and Ctrl-C never write a task.
- No new environment variables or task syntax are introduced.

## File Map

- Create: `internal/cli/add_options.go` — action-local option and phase parser.
- Create: `internal/cli/add_options_test.go` — parser and usage-contract tests.
- Create: `internal/cli/add_candidates.go` — configured-file candidate collector and metadata model.
- Create: `internal/cli/add_candidates_test.go` — union, deduplication, ordering, and missing-file tests.
- Create: `internal/cli/add_terminal.go` — readline completer, selector state machine, TTY prompts, and line-oriented fallback.
- Create: `internal/cli/add_terminal_test.go` — completer and selector state tests.
- Create: `internal/cli/add_guided.go` — guided flow, metadata parsing, and final task composition.
- Create: `internal/cli/add_guided_test.go` — guided composition and scripted-input tests.
- Modify: `internal/cli/add.go` — parse action options and dispatch interactive/guided flows before the existing add path.
- Modify: `internal/cli/actions.go` — keep one buffered stdin reader per session and wire the input boundary.
- Modify: `internal/cli/testdata/script/help.txtar` — update the pinned add help block.
- Modify: `internal/cli/testdata/script/t2300-interactive-add.txtar` — add end-to-end scripted mode coverage.
- Modify: `README.md` — document both modes, phase selectors, candidate sources, and the non-TTY protocol.
- Modify: `man/gtdo.1.tmpl` — document interactive add behavior and configured task-bearing files.
- Modify: `man/man_test.go` — pin the new manual section and option names.
- Modify: `go.mod`, `go.sum` — add the readline runtime dependency.
- Regenerate: `man/gtdo.1` with `make man`.

---

### Task 1: Parse action-local add modes

**Files:**
- Create: `internal/cli/add_options.go`
- Test: `internal/cli/add_options_test.go`
- Modify: `internal/cli/add.go` only after the parser tests are green

**Interfaces:**
- Produces `type addMode uint8` with `addModeNone`, `addModeInteractive`, and `addModeGuided`.
- Produces `type guidedPhase string` with constants `metadata`, `project`, and `context`.
- Produces:

  ```go
  type addOptions struct {
      Mode       addMode
      Only       map[guidedPhase]bool
      Positional []string
  }

  func parseAddOptions(args []string) (addOptions, error)
  func (o addOptions) phaseEnabled(phase guidedPhase) bool
  func addUsage() string
  ```

- Recognizes exact leading options `-i`, `--interactive`, `-g`, `--guided`, `--only phase`, and `--only=phase`; after the first positional argument, all remaining words are positional text.
- Rejects both modes together, `--only` without guided mode, unknown phase names, missing phase values, and positional text with either explicit mode.
- Leaves unrecognized legacy words such as `-task` positional when no recognized mode option has started.

- [ ] **Step 1: Write the failing parser tests.**

  Add table-driven cases that assert the exact parsed mode and phase map:

  ```go
  func TestParseAddOptions(t *testing.T) {
      tests := []struct {
          name string
          args []string
          mode addMode
          only []guidedPhase
          text []string
          wantErr bool
      }{
          {name: "legacy text", args: []string{"buy", "milk"}, text: []string{"buy", "milk"}},
          {name: "interactive shorthand", args: []string{"-i"}, mode: addModeInteractive},
          {name: "guided long repeatable phases", args: []string{"--guided", "--only", "project", "--only=context"}, mode: addModeGuided, only: []guidedPhase{phaseProject, phaseContext}},
          {name: "mode with text", args: []string{"-g", "task"}, wantErr: true},
          {name: "only without guided", args: []string{"--only", "context"}, wantErr: true},
          {name: "both modes", args: []string{"-i", "--guided"}, wantErr: true},
          {name: "unknown phase", args: []string{"-g", "--only", "tags"}, wantErr: true},
      }
      // Compare mode, sorted phase membership, and positional text.
  }
  ```

  Add `TestAddUsage` to pin that `addUsage()` names `-i`, `--interactive`, `-g`, `--guided`, and `--only`.

- [ ] **Step 2: Run the focused tests and verify the expected failure.**

  Run:

  ```bash
  go test ./internal/cli -run 'Test(ParseAddOptions|AddUsage)' -count=1
  ```

  Expected: compilation failure because the parser types and functions do not exist.

- [ ] **Step 3: Implement the minimal parser.**

  Scan from the first argument until the first non-option. For `--only`, consume the next word or split `--only=value`; map the value to one of the three constants. On every invalid combination return a plain error; the action layer will render `addUsage()` through the existing usage failure path. `phaseEnabled` returns all three phases when `Only` is empty and otherwise checks the map.

- [ ] **Step 4: Run parser tests and the existing CLI package tests.**

  Run:

  ```bash
  gofmt -w internal/cli/add_options.go internal/cli/add_options_test.go
  go test ./internal/cli -run 'Test(ParseAddOptions|AddUsage)' -count=1
  go test ./internal/cli -count=1
  ```

  Expected: PASS; no existing testscript output changes yet because `actionAdd` has not been wired.

- [ ] **Step 5: Commit the parser.**

  ```bash
  git add internal/cli/add_options.go internal/cli/add_options_test.go
  git commit -m "feat(cli): parse interactive add modes"
  ```

### Task 2: Collect candidates from configured task files

**Files:**
- Create: `internal/cli/add_candidates.go`
- Test: `internal/cli/add_candidates_test.go`

**Interfaces:**
- Produces:

  ```go
  type metadataCandidate struct {
      Key    string
      Values []string
  }

  type addCandidates struct {
      Contexts []string
      Projects []string
      Metadata []metadataCandidate
  }

  func collectAddCandidates(cfg *config.Config) addCandidates
  func collectAddCandidatesFromPaths(paths []string) addCandidates
  ```

- Reads each unique path once, ignores read errors, uses `todo.SigilWords` for `@` and `+`, and extracts metadata words matching the existing formatter rule `^[A-Za-z0-9]+:[^ \t]+$`.
- Splits metadata at the first colon, deduplicates keys and values case-sensitively, and sorts contexts, projects, keys, and each key's values with `sort.Strings`.
- `collectAddCandidates` passes exactly `cfg.TodoFile` and `cfg.DoneFile`; it never passes `cfg.ReportFile`.

- [ ] **Step 1: Write the failing collector tests.**

  Use two temporary files with overlapping and unique data, plus a report file that contains a decoy sigil. Assert the sorted union and metadata grouping:

  ```go
  func TestCollectAddCandidatesUnionsTaskFiles(t *testing.T) {
      dir := t.TempDir()
      todoPath := filepath.Join(dir, "todo.txt")
      donePath := filepath.Join(dir, "done.txt")
      reportPath := filepath.Join(dir, "report.txt")
      if err := os.WriteFile(todoPath, []byte("one @home +gtdo due:today\ntwo @work +personal\n"), 0o644); err != nil { t.Fatal(err) }
      if err := os.WriteFile(donePath, []byte("done @home +archive due:yesterday status:done\n"), 0o644); err != nil { t.Fatal(err) }
      if err := os.WriteFile(reportPath, []byte("report @decoy +decoy\n"), 0o644); err != nil { t.Fatal(err) }

      got := collectAddCandidates(&config.Config{TodoFile: todoPath, DoneFile: donePath, ReportFile: reportPath})
      // Assert Contexts == []string{"@home", "@work"}, Projects == []string{"+archive", "+gtdo", "+personal"}, and sorted metadata.
  }
  ```

  Add tests for duplicate paths, missing files, malformed metadata, and byte-wise case-sensitive ordering.

- [ ] **Step 2: Run the collector tests and verify the expected failure.**

  Run:

  ```bash
  go test ./internal/cli -run TestCollectAddCandidates -count=1
  ```

  Expected: compilation failure because `addCandidates` and the collectors do not exist.

- [ ] **Step 3: Implement the collector.**

  Deduplicate input paths before reading. For every line, call `todo.SigilWords([]string{line}, '@', nil)` and the equivalent `+` call, then walk `strings.FieldsFunc` tokens and group valid metadata words by their first-colon key. Treat a nil or empty file as a normal no-candidate result.

- [ ] **Step 4: Run collector and domain tests.**

  Run:

  ```bash
  gofmt -w internal/cli/add_candidates.go internal/cli/add_candidates_test.go
  go test ./internal/cli -run TestCollectAddCandidates -count=1
  go test ./internal/todo ./internal/cli -count=1
  ```

  Expected: PASS, with no changes to existing completion behavior.

- [ ] **Step 5: Commit candidate collection.**

  ```bash
  git add internal/cli/add_candidates.go internal/cli/add_candidates_test.go
  git commit -m "feat(cli): collect add completion candidates"
  ```

### Task 3: Add readline completion and selector state

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/cli/add_terminal.go`
- Test: `internal/cli/add_terminal_test.go`

**Interfaces:**
- Adds direct dependency `github.com/chzyer/readline v1.5.1`.
- Produces:

  ```go
  type addCompleter struct{ Candidates addCandidates }

  func (c addCompleter) Do(line []rune, pos int) ([][]rune, int)

  type selectorModel struct {
      Options  []string
      Selected map[string]bool
      Cursor   int
      Query    string
  }

  func newSelectorModel(options []string) selectorModel
  func (m *selectorModel) Move(delta int)
  func (m *selectorModel) Toggle()
  func (m *selectorModel) SetQuery(query string)
  func (m selectorModel) Visible() []string
  func (m selectorModel) Values() []string
  ```

- The completer finds the whitespace-delimited token containing the cursor. It offers only contexts for a token beginning `@` and projects for a token beginning `+`, returning readline suffixes plus the number of already-consumed runes. Other words return no candidates.
- The selector model preserves sorted option order, keeps cursor bounds valid after filtering, toggles by exact token, and returns selected values sorted. It is independent of terminal I/O.

- [ ] **Step 1: Add the dependency and write failing pure tests.**

  Add the module requirement with:

  ```bash
  go get github.com/chzyer/readline@v1.5.1
  ```

  Then add tests such as:

  ```go
  func TestAddCompleterUsesCurrentSigilWord(t *testing.T) {
      c := addCompleter{Candidates: addCandidates{Contexts: []string{"@home", "@office"}, Projects: []string{"+gtdo"}}}
      got, consumed := c.Do([]rune("Call @o"), len([]rune("Call @o")))
      // Assert consumed == 2 and got contains []rune("ffice") only.
  }

  func TestSelectorModelFiltersAndToggles(t *testing.T) {
      m := newSelectorModel([]string{"@home", "@office", "@phone"})
      m.SetQuery("of")
      m.Toggle()
      if got := m.Values(); !slices.Equal(got, []string{"@office"}) { t.Fatal(got) }
  }
  ```

  Include tests for `+` prefixes, empty/no-match prefixes, cursor movement, query clearing, and stable selected ordering.

- [ ] **Step 2: Run the focused tests and verify the expected failure.**

  Run:

  ```bash
  go test ./internal/cli -run 'Test(AddCompleter|SelectorModel)' -count=1
  ```

  Expected: compilation failure because the completer and selector model do not exist.

- [ ] **Step 3: Implement the completer and selector model.**

  Compute the token start by walking backward from `pos` to a space or tab. Use the typed token's first rune to select the candidate slice, filter with `strings.HasPrefix`, compute the longest shared suffix after the typed prefix, and return the candidate suffixes in sorted order. Implement selector filtering with a case-sensitive substring query, clamp the cursor after every query/move, and return selected options in their original sorted order.

- [ ] **Step 4: Run focused tests and verify dependency hygiene.**

  Run:

  ```bash
  gofmt -w internal/cli/add_terminal.go internal/cli/add_terminal_test.go
  go test ./internal/cli -run 'Test(AddCompleter|SelectorModel)' -count=1
  go mod tidy
  go test ./internal/cli -count=1
  ```

  Expected: PASS; `go.mod` lists readline as a direct dependency and no unrelated module changes appear.

- [ ] **Step 5: Commit terminal primitives.**

  ```bash
  git add go.mod go.sum internal/cli/add_terminal.go internal/cli/add_terminal_test.go
  git commit -m "feat(cli): add terminal completion primitives"
  ```

### Task 4: Build guided composition and scripted input

**Files:**
- Create: `internal/cli/add_guided.go`
- Test: `internal/cli/add_guided_test.go`
- Modify: `internal/cli/actions.go` to retain one buffered reader per session

**Interfaces:**
- Produces:

  ```go
  type addInput interface {
      PromptTask(addCandidates) (string, error)
      PromptMetadata(addCandidates) ([]string, error)
      Select(guidedPhase, []string) ([]string, error)
  }

  func runGuided(input addInput, candidates addCandidates, opts addOptions) (string, error)
  func composeGuidedTask(base string, metadata, projects, contexts []string) string
  ```

- `runGuided` always calls `PromptTask`, then calls enabled phases in metadata/project/context order. It appends accepted metadata pairs as `key:value`, projects, and contexts; it skips duplicate sigil tokens already present in `base`.
- A `lineAddInput` implementation consumes the non-TTY protocol: one task line; zero or more complete `key:value` lines terminated by an empty line; one space-separated project line; one space-separated context line. It filters project/context selections to the supplied candidate list and treats empty lines as empty selections.
- Change `session` to hold `reader *bufio.Reader`, initialize it in `newSession`, and have `readLine`/a new `readLineErr` use that same reader so multi-phase piped input cannot be over-read and discarded.

- [ ] **Step 1: Write failing guided-flow and reader tests.**

  Add a fake `addInput` that records phase calls and returns deterministic values:

  ```go
  func TestRunGuidedRunsSelectedPhasesInOrder(t *testing.T) {
      input := &fakeAddInput{
          task: "Call team @home",
          metadata: []string{"due:tomorrow"},
          selections: map[guidedPhase][]string{
              phaseProject: {"+gtdo"},
              phaseContext: {"@phone", "@home"},
          },
      }
      got, err := runGuided(input, addCandidates{}, addOptions{Mode: addModeGuided})
      if err != nil { t.Fatal(err) }
      want := "Call team @home due:tomorrow +gtdo @phone"
      if got != want { t.Fatalf("got %q, want %q", got, want) }
      if !slices.Equal(input.calls, []guidedPhase{phaseMetadata, phaseProject, phaseContext}) { t.Fatal(input.calls) }
  }
  ```

  Add a selective-phase test for `--only context`, duplicate-token tests, empty-selection tests, malformed metadata-line tests, and a session reader test that reads three successive lines from one `strings.Reader`.

- [ ] **Step 2: Run the focused tests and verify the expected failure.**

  Run:

  ```bash
  go test ./internal/cli -run 'Test(RunGuided|ComposeGuided|SessionReader)' -count=1
  ```

  Expected: compilation failure because the guided flow, input interface, and persistent reader do not exist.

- [ ] **Step 3: Implement composition and line input.**

  Use `todo.Task{Text: base}.Contexts()` and `.Projects()` to detect exact existing sigil words. For metadata, split each scripted line at its first colon and require a non-empty alphanumeric key and value; return an error for a non-empty malformed line. In `composeGuidedTask`, append each non-empty group with one space only when the accumulated text is non-empty.

- [ ] **Step 4: Run guided tests and all existing CLI tests.**

  Run:

  ```bash
  gofmt -w internal/cli/add_guided.go internal/cli/add_guided_test.go internal/cli/actions.go
  go test ./internal/cli -run 'Test(RunGuided|ComposeGuided|SessionReader)' -count=1
  go test ./internal/cli -count=1
  ```

  Expected: PASS; existing add/append/delete prompt tests remain byte-identical.

- [ ] **Step 5: Commit guided composition and line fallback.**

  ```bash
  git add internal/cli/add_guided.go internal/cli/add_guided_test.go internal/cli/actions.go
  git commit -m "feat(cli): compose guided add input"
  ```

### Task 5: Wire TTY input and the add action

**Files:**
- Modify: `internal/cli/add_terminal.go` — TTY `readline` adapter and raw selector loop.
- Modify: `internal/cli/add.go` — mode parsing, candidate collection, dispatch, cancellation, and final store add.
- Modify: `internal/cli/actions.go` — construct the input adapter and retain the session reader.
- Modify: `internal/cli/actions.go` or `internal/cli/help.go` — add action help text.

**Interfaces:**
- `newAddInput(s *session, candidates addCandidates) addInput` chooses `ttyAddInput` only when `stdinIsTTY(s.in)` and the input is an `*os.File`; otherwise it returns `lineAddInput`.
- `ttyAddInput.PromptTask` and metadata prompts use `readline.NewEx` with `readline.AutoCompleter`; the project/context selector uses `readline.NewTerminal` in raw mode and maps `readline.CharPrev`/`CharNext` to Up/Down, Space to toggle, `/` to query mode, Enter to confirm, and Esc to clear/exit.
- `actionAdd` parses before `newSession`; parser errors print `addUsage()` directly to `cmd.ErrOrStderr()` so invalid modes do not trigger file ensuring. For `addModeNone`, the existing `addInput` path and output are unchanged. For interactive/guided modes, it collects candidates, obtains one final text, calls `Store.Add` once, and calls `printAdded` exactly as before.
- Handle `io.EOF` and `*readline.InterruptError` from explicit modes by returning `exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)` without writing stderr or the todo file. Preserve the existing `-f` behavior for legacy adds; reject `-f` combined with explicit modes through `addUsage()` so an explicit interactive request is never silently skipped.

- [ ] **Step 1: Write failing action tests.**

  Add unit tests with a temporary config/session for `actionAdd` or its dispatch helper that assert:

  ```go
  func TestActionAddGuidedUsesStoreOnce(t *testing.T) { /* fake input returns task and phases; assert one final Add result */ }
  func TestActionAddModeErrorsDoNotEnsureFiles(t *testing.T) { /* invalid args; assert usage error and absent todo path */ }
  ```

  Add a test that `newAddInput` chooses the line adapter for a `strings.Reader`, and a selector-state test for Esc/Enter behavior. Keep TTY-specific byte rendering out of unit assertions.

- [ ] **Step 2: Run the focused action tests and verify the expected failure.**

  Run:

  ```bash
  go test ./internal/cli -run 'TestActionAdd|TestNewAddInput' -count=1
  ```

  Expected: failure because mode dispatch and adapters are not wired.

- [ ] **Step 3: Implement the TTY adapter and action dispatch.**

  Create each readline instance with `Prompt: "Add: "` (or the phase-specific prompt), `Stdin` as the terminal file, and `Stdout`/`Stderr` as the session error writer so prompts remain on stderr like existing TTY prompts. Use `DisableAutoSaveHistory` and an empty history path so an add never creates a history file. The raw selector prints the help line, redraws the filtered list after each key, and restores terminal mode with `defer` on every exit path.

  Change `actionAdd` to the following control shape:

  ```go
  opts, err := parseAddOptions(args)
  if err != nil || (cfg.Force && opts.Mode != addModeNone) {
      fmt.Fprintln(cmd.ErrOrStderr(), addUsage())
      return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
  }
  s, err := newSession(cmd, cfg)
  if err != nil { return err }
  if opts.Mode == addModeNone {
      return addLegacy(s, opts.Positional)
  }
  candidates := collectAddCandidates(cfg)
  input := newAddInput(s, candidates)
  var text string
  if opts.Mode == addModeInteractive { text, err = input.PromptTask(candidates) } else { text, err = runGuided(input, candidates, opts) }
  if err != nil { return cancelOrDie(s, err) }
  line, finalText, err := s.store.Add(text, cfg.DateOnAdd, cfg.PriorityOnAdd, now())
  if err != nil { return s.die(err.Error()) }
  return s.printAdded(s.store.TodoFile, line, finalText)
  ```

  Keep `addLegacy` byte-for-byte equivalent to the current `actionAdd` body and retain its existing usage string.

- [ ] **Step 4: Run action tests and the existing session suite.**

  Run:

  ```bash
  gofmt -w internal/cli/add.go internal/cli/add_terminal.go internal/cli/actions.go
  go test ./internal/cli -run 'TestActionAdd|TestNewAddInput' -count=1
  go test ./internal/cli -count=1
  ```

  Expected: PASS; only new mode tests exercise the new code, and all existing script sessions remain green.

- [ ] **Step 5: Commit the wired add modes.**

  ```bash
  git add internal/cli/add.go internal/cli/add_terminal.go internal/cli/actions.go
  git commit -m "feat(cli): add interactive and guided modes"
  ```

### Task 6: Pin scripted CLI behavior

**Files:**
- Create: `internal/cli/testdata/script/t2300-interactive-add.txtar`
- Modify: `internal/cli/testdata/script/help.txtar` — update the pinned `help.want` add block for the new long help text.

**Interfaces:**
- The testscript is the byte-parity contract for non-TTY mode; it must assert stdout, stderr, exit status, and exact todo/done file bytes.

- [ ] **Step 1: Write the failing testscript.**

  Add fixtures and sessions with both task files:

  ```text
  # candidates are a union of todo.txt and done.txt; report.txt is a decoy
  -- todo.txt --
  current @home +gtdo due:today
  -- done.txt --
  old @phone +legacy status:done
  -- report.txt --
  report @ignored +ignored

  exec gtdo add -i
  stdin interactive.in
  cmp stdout interactive.want
  cmp todo.txt interactive.todo.want

  exec gtdo add -g
  stdin guided.in
  cmp stdout guided.want
  cmp todo.txt guided.todo.want

  exec gtdo add -g --only context
  stdin context-only.in
  cmp stdout context-only.want
  cmp todo.txt context-only.todo.want

  ! exec gtdo add -i extra text
  stdout '^$'
  stderr '^usage: gtdo add'
  ```

  Use `interactive.in` with a task containing `@phone +legacy`, and `guided.in` with the non-TTY protocol: task line, metadata line, blank metadata terminator, project line, context line. Assert duplicate `@home` is not appended and the final confirmation line number/text are exact.

- [ ] **Step 2: Run the new script and verify it fails.**

  Run:

  ```bash
  go test ./internal/cli -run TestScript -count=1 -v
  ```

  Expected: the new script fails before the mode implementation is complete or because its expected files do not exist.

- [ ] **Step 3: Add exact wants and correct only implementation mismatches.**

  Pin the expected final lines using the existing verbose add format (`N text` followed by `TODO: N added.`), keep stderr empty for successful piped input, and include no prompt text. Pin invalid-mode stderr and exit status 1. Do not relax comparisons with regular expressions when a literal byte comparison is possible.

- [ ] **Step 4: Run the focused and complete script suite.**

  Run:

  ```bash
  go test ./internal/cli -run TestScript -count=1
  go test ./internal/cli -count=1
  ```

  Expected: PASS for the new sessions and every existing txtar parity case.

- [ ] **Step 5: Commit the scripted contract.**

  ```bash
  git add internal/cli/testdata/script/t2300-interactive-add.txtar internal/cli/testdata/script/help.txtar
  git commit -m "test(cli): cover interactive add modes"
  ```

### Task 7: Document the commands and regenerate the man page

**Files:**
- Modify: `README.md`
- Modify: `man/gtdo.1.tmpl`
- Modify: `man/man_test.go`
- Modify: `internal/cli/testdata/script/help.txtar` — retain the updated help golden file from Task 6.
- Regenerate: `man/gtdo.1` with `make man`.

**Interfaces:**
- README and man page document the exact invocations `gtdo add -i`, `gtdo add -g`, repeatable `--only`, candidate sources (`todo_file`/`done_file` only), selector controls, and the line-oriented non-TTY protocol.
- The manual test additionally requires `.B -i`, `.B -g`, and `.B --only` plus the phrase that candidates come from the configured todo and done files.

- [ ] **Step 1: Write the failing manual assertion.**

  Extend `TestManPageHasTheSectionsCobraDoesNotEmit` with:

  ```go
  for _, text := range []string{"\\-i", "\\-g", "--only", "todo_file", "done_file"} {
      if !strings.Contains(string(data), text) {
          t.Errorf("man/gtdo.1 is missing %q", text)
      }
  }
  ```

  Run `go test ./man -run TestManPageHasTheSectionsCobraDoesNotEmit -count=1` and confirm it fails before documentation changes.

- [ ] **Step 2: Update user-facing documentation.**

  Add README examples and a concise “Interactive add” subsection. Add a man-template paragraph under DESCRIPTION and a `.SH INTERACTIVE ADD` section that lists the command forms, phase names, controls, source-file rules, and non-TTY fallback. Expand the add action’s Long text so `gtdo help add` advertises the new flags without changing its Short line.

- [ ] **Step 3: Regenerate and verify the committed page.**

  Run:

  ```bash
  make man
  go test ./man -count=1
  go test ./internal/cli -run TestScript -count=1
  ```

  Update `help.want` only for the intentional add Long-text change; keep all unrelated help bytes unchanged.

- [ ] **Step 4: Commit documentation.**

  ```bash
  git add README.md man/gtdo.1.tmpl man/gtdo.1 man/man_test.go internal/cli/actions.go internal/cli/testdata/script/help.txtar
  git commit -m "docs: document interactive add modes"
  ```

### Task 8: Full verification and handoff

**Files:**
- No new files; verify the complete working tree and generated artifacts.

- [ ] **Step 1: Run formatting and the full test suite.**

  ```bash
  gofmt -w internal/cli/*.go
  go test ./...
  ```

  Expected: PASS with no testscript diffs.

- [ ] **Step 2: Run repository build and lint checks.**

  ```bash
  make build
  make lint
  ```

  Expected: `./gtdo` builds with version metadata and golangci-lint reports no findings.

- [ ] **Step 3: Verify generated documentation and repository state.**

  ```bash
  make man
  git status --short
  git diff --check
  ```

  Expected: the generated man page is current, no unintended files are modified, and the diff has no whitespace errors. Remove only the generated local `./gtdo` binary if it is untracked; do not remove user files.

- [ ] **Step 4: Report completion with evidence.**

  Summarize the new invocations, candidate-source rule, test commands, and any platform limitation of TTY rendering. Do not claim completion until `go test ./...`, `make build`, `make lint`, and `make man` all succeed.
