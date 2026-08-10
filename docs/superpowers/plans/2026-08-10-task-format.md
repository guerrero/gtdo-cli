# Configurable Task Format Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gtdo format [FILE]` to rewrite configured todo files using a validated, configurable order of checked state, priority, UUID, content, keywords, projects, and contexts.

**Architecture:** Keep field parsing and template rendering pure in `internal/todo`, with a `TaskFormat` value parsed once from `Config.TaskFormat`. Add a file reformatter that returns rewritten bytes while reusing the existing line/newline model, then let the CLI pre-read all targets and write them only after validation succeeds. Add the action through the existing Cobra action-spec path so help, shorthelp, completion, and the generated man page stay derived from one registration.

**Tech Stack:** Go 1.26.5, Cobra, BurntSushi/toml, go-internal testscript, existing `internal/todo` line store and regex helpers.

## Global Constraints

- The default configuration is `task_format = "[checked][priority][uuid][content][keywords][project][context]"` under `[behavior]`.
- The formatter recognizes only `[checked]`, `[priority]`, `[uuid]`, `[content]`, `[keywords]`, `[project]`, and `[context]` placeholders, with optional whitespace between placeholders and no literal text.
- It inserts one ASCII space between adjacent non-empty fields; absent fields create no leading, trailing, or doubled separator spaces.
- `[checked]` emits `x` only for a completed line and emits `""` otherwise; an `x YYYY-MM-DD` completion date is consumed and never emitted.
- `[priority]` emits the existing `(A)`-style priority, `[uuid]` recognizes `YYYYMMDDTHHMMSS.ssZ`, `[keywords]` recognizes `key:value` words, and project/context extraction keeps the existing `+name`/`@name` rules.
- `[content]` is every remaining word in source order; repeated values within any field retain source order and are separated by one space.
- Blank lines and the source file's trailing-newline state are preserved.
- `gtdo format` with no argument rewrites configured `todo.txt` and `done.txt`; one relative argument resolves under the configured TODO directory, and one absolute argument is accepted.
- The template is validated and all selected files are read/formatted before any selected file is written.
- An explicit target must be an existing regular file; malformed templates, missing targets, directories, and too many arguments fail without rewriting task content.
- Successful formatting prints `TODO: <file> formatted.` per target when verbosity is enabled and prints nothing when `TODOTXT_VERBOSE=0`.
- `task_format` is TOML-only; it has no environment-variable or global-flag override.
- Existing list, filter, sort, mutation, prompt, exit-code, and output behavior remains unchanged outside the new action/configuration.
- Every implementation change follows red-green-refactor: write a failing test, run it and observe the expected feature failure, implement the smallest change, run the focused test, then run the relevant package suite.
- Use Conventional Commit subjects with the repository's allowed scopes; each completed task ends with its own commit.

---

## File Map

- Modify `internal/config/defaults.go` to define the canonical default template.
- Modify `internal/config/toml.go` to decode `[behavior].task_format`.
- Modify `internal/config/config.go` to expose the resolved `Config.TaskFormat`.
- Modify `internal/config/config_test.go` to pin the default and TOML override.
- Create `internal/todo/task_format.go` for the template parser, field extraction, line rendering, and read-only file reformatter.
- Create `internal/todo/task_format_test.go` for parser, extraction, rendering, and newline-preservation tests.
- Modify `internal/todo/store.go` to factor its existing line serialization into a reusable byte helper.
- Create `internal/cli/format.go` for `format [FILE]`, target resolution, validation, pre-read/write flow, and success messages.
- Modify `internal/cli/root.go` to register the new action.
- Create `internal/cli/testdata/script/t2300-format.txtar` for both-file, single-file, custom-order, quiet, invalid, and target-validation sessions.
- Modify existing help/default-action testscript fixtures whose exact action lists gain `format`.
- Modify `README.md`, `CHANGELOG.md`, and `man/gtdo.1.tmpl`; regenerate `man/gtdo.1` with `make man`.

## Interfaces

The implementation plan uses these exact interfaces between tasks:

```go
// internal/config/defaults.go
const DefaultTaskFormat = "[checked][priority][uuid][content][keywords][project][context]"

// internal/config/config.go
type Config struct {
    // existing fields...
    TaskFormat string
}

// internal/todo/task_format.go
type TaskFormat struct { /* parsed fields remain private */ }

func ParseTaskFormat(spec string) (TaskFormat, error)
func (f TaskFormat) FormatLine(line string) string
func ReformatFile(path string, f TaskFormat) ([]byte, error)
```

`ReformatFile` is deliberately read-only. The CLI collects its returned bytes
for every target before calling `os.WriteFile`, which gives the command the
required preflight behavior without broadening the existing `Store` mutation
API.

## Task 1: Add the configuration value

**Files:**

- Modify: `internal/config/defaults.go`
- Modify: `internal/config/toml.go`
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**

- Produces `config.DefaultTaskFormat` and `config.Config.TaskFormat` for the formatter action.
- Does not add an environment variable or CLI option; TOML is the only override layer.

- [ ] **Step 1: Write the failing default-config test.**

Add an assertion to `TestDefaults` immediately after the existing behavior
assertions:

```go
if cfg.TaskFormat != DefaultTaskFormat {
    t.Errorf("TaskFormat = %q, want %q", cfg.TaskFormat, DefaultTaskFormat)
}
```

Add a dedicated TOML test so the string is covered independently of the
existing bool/string precedence table:

```go
func TestTaskFormatFromTOML(t *testing.T) {
    h := home(t)
    body := "[behavior]\ntask_format = \"[project][content][keywords][context]\"\n"
    cfg := loadWith(t, withOpts(t, Options{}, body), h)
    want := "[project][content][keywords][context]"
    if cfg.TaskFormat != want {
        t.Errorf("TaskFormat = %q, want %q", cfg.TaskFormat, want)
    }
}
```

- [ ] **Step 2: Run the focused config tests and verify they fail for the missing field.**

Run:

```bash
go test ./internal/config -run 'TestDefaults|TestTaskFormatFromTOML' -count=1
```

Expected: compilation/test failure because `DefaultTaskFormat` and
`Config.TaskFormat` do not exist yet.

- [ ] **Step 3: Add the default and TOML/resolution wiring.**

In `internal/config/defaults.go`, define:

```go
const DefaultTaskFormat = "[checked][priority][uuid][content][keywords][project][context]"
```

Set `TaskFormat: DefaultTaskFormat` in the `behaviorTOML` returned by
`defaultFileConfig`. Add `TaskFormat string \`toml:"task_format"\`` to
`behaviorTOML`. Add `TaskFormat string` to `Config` near the other behavior
strings, and set `TaskFormat: f.Behavior.TaskFormat` in `resolve`. Do not call
`pickString` or consult an environment variable, because this setting is
intentionally TOML-only and the default is already present before decoding.

- [ ] **Step 4: Run the focused tests and the full config package.**

Run:

```bash
go test ./internal/config -run 'TestDefaults|TestTaskFormatFromTOML' -count=1
go test ./internal/config -count=1
```

Expected: both commands pass with no warnings.

- [ ] **Step 5: Commit the configuration change.**

```bash
git add internal/config/defaults.go internal/config/toml.go internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add task format setting"
```

## Task 2: Implement the pure template parser and line formatter

**Files:**

- Create: `internal/todo/task_format.go`
- Test: `internal/todo/task_format_test.go`

**Interfaces:**

- Consumes the default/custom string supplied by `config.Config.TaskFormat`.
- Produces `todo.TaskFormat`, `ParseTaskFormat`, and `TaskFormat.FormatLine`.
- Does not read or write files and does not change the existing `Task` parsing
  methods used by list and mutation actions.

- [ ] **Step 1: Write failing tests for template parsing and all field rules.**

Create `internal/todo/task_format_test.go` with these helpers and cases:

```go
package todo

import (
    "strings"
    "testing"
)

const defaultTaskFormat = "[checked][priority][uuid][content][keywords][project][context]"

func mustTaskFormat(t *testing.T, spec string) TaskFormat {
    t.Helper()
    f, err := ParseTaskFormat(spec)
    if err != nil {
        t.Fatalf("ParseTaskFormat(%q): %v", spec, err)
    }
    return f
}

func TestTaskFormatDefaultOrder(t *testing.T) {
    f := mustTaskFormat(t, defaultTaskFormat)
    got := f.FormatLine("(B) write report key:one +work @desk 20260808T143045.12Z")
    want := "(B) 20260808T143045.12Z write report key:one +work @desk"
    if got != want {
        t.Errorf("FormatLine = %q, want %q", got, want)
    }
}

func TestTaskFormatCustomOrderAndChecked(t *testing.T) {
    f := mustTaskFormat(t, "[project][content][keywords][context][uuid][priority][checked]")
    got := f.FormatLine("x 2026-08-08 finish key:done +archive @desk 20260808T143045.12Z")
    want := "+archive finish key:done @desk 20260808T143045.12Z x"
    if got != want {
        t.Errorf("FormatLine = %q, want %q", got, want)
    }
}

func TestTaskFormatOmitsMissingFieldsWithoutExtraSpaces(t *testing.T) {
    f := mustTaskFormat(t, defaultTaskFormat)
    if got := f.FormatLine("plain task"); got != "plain task" {
        t.Errorf("FormatLine = %q, want %q", got, "plain task")
    }
    if got := f.FormatLine("x 2026-08-08"); got != "x" {
        t.Errorf("checked-only FormatLine = %q, want %q", got, "x")
    }
}

func TestTaskFormatKeepsRepeatedValuesInSourceOrder(t *testing.T) {
    f := mustTaskFormat(t, defaultTaskFormat)
    line := "task key:first +one @home key:second +two @work 20260808T143045.12Z"
    want := "20260808T143045.12Z task key:first key:second +one +two @home @work"
    if got := f.FormatLine(line); got != want {
        t.Errorf("FormatLine = %q, want %q", got, want)
    }
}

func TestTaskFormatLeavesInvalidUUIDAsContent(t *testing.T) {
    f := mustTaskFormat(t, defaultTaskFormat)
    line := "task 20260808T143045.1Z 20260808T143045.12Z"
    want := "20260808T143045.12Z task 20260808T143045.1Z"
    if got := f.FormatLine(line); got != want {
        t.Errorf("FormatLine = %q, want %q", got, want)
    }
}

func TestParseTaskFormatRejectsMalformedOrUnknownFields(t *testing.T) {
    for _, spec := range []string{"", "[unknown]", "[content", "content", "[content]literal"} {
        t.Run(spec, func(t *testing.T) {
            if _, err := ParseTaskFormat(spec); err == nil || !strings.Contains(err.Error(), "invalid task format") {
                t.Fatalf("ParseTaskFormat(%q) error = %v, want invalid task format", spec, err)
            }
        })
    }
}
```

These tests intentionally include checked-date removal, automatic separators,
custom ordering, all requested classifiers, repeated values, invalid UUID
fallback, and malformed templates.

- [ ] **Step 2: Run the formatter tests and verify the expected red failure.**

Run:

```bash
go test ./internal/todo -run 'TestTaskFormat' -count=1
```

Expected: compilation failure because `TaskFormat` and `ParseTaskFormat` do
not exist yet.

- [ ] **Step 3: Implement the parser and formatter minimally.**

In `internal/todo/task_format.go`, define a private enum for the seven field
names and a `TaskFormat` containing `fields []taskFormatField`. Parse only
bracketed field names, skipping ASCII spaces, tabs, and newlines between
placeholders. Reject empty specifications, unknown names, missing closing
brackets, and any non-whitespace text outside a placeholder with errors whose
messages begin `invalid task format:`.

Implement `FormatLine` with this exact extraction order:

1. Split with `strings.Fields`.
2. If the first word is exactly `x`, set checked to `x`, remove it, and remove
   the next word only when it matches the full `^(19|20)[0-9]{2}-[0-9]{2}-[0-9]{2}$`
   completion-date shape.
3. If the next word is an uppercase priority word matching
   `^\([A-Z]\)$`, remove it as priority. This accepts a priority after the
   checked prefix while leaving existing `Task.Priority` semantics untouched.
4. Classify each remaining word in source order: exact UUID
   `^[0-9]{8}T[0-9]{6}\.[0-9]{2}Z$`, existing `metaWordRe` for keywords,
   existing `projectWordRe`, existing `contextWordRe`, and content fallback.
5. Render a requested field as its string or the space-joined values for that
   field. Append only non-empty rendered fields, then `strings.Join(parts, " ")`.

Keep `[checked]` as the literal `x` value or an empty value; do not emit a
trailing separator for a checked-only line.

- [ ] **Step 4: Run the formatter tests and package suite.**

Run:

```bash
gofmt -w internal/todo/task_format.go internal/todo/task_format_test.go
go test ./internal/todo -run 'TestTaskFormat' -count=1
go test ./internal/todo -count=1
```

Expected: all formatter tests and all existing todo tests pass.

- [ ] **Step 5: Commit the pure formatter.**

```bash
git add internal/todo/task_format.go internal/todo/task_format_test.go
git commit -m "feat(todo): add configurable task line formatter"
```

## Task 3: Add read-only file reformatting with newline preservation

**Files:**

- Modify: `internal/todo/store.go`
- Modify: `internal/todo/task_format.go`
- Test: `internal/todo/task_format_test.go`

**Interfaces:**

- Consumes `TaskFormat.FormatLine` from Task 2.
- Produces `ReformatFile(path string, f TaskFormat) ([]byte, error)` for the
  CLI preflight phase.
- Keeps all writing responsibility in the caller; this function must never
  mutate its input file.

- [ ] **Step 1: Write failing file and byte-shape tests.**

Append these tests to `internal/todo/task_format_test.go`, using the existing
`writeFile` and `readFile` helpers where useful:

```go
func TestReformatFilePreservesBlankLinesAndTrailingNewline(t *testing.T) {
    path := filepath.Join(t.TempDir(), "todo.txt")
    writeFile(t, path, "(A) task +project\n\nkey:value other\n")
    f := mustTaskFormat(t, defaultTaskFormat)

    got, err := ReformatFile(path, f)
    if err != nil {
        t.Fatal(err)
    }
    want := "(A) task +project\n\nkey:value other\n"
    if string(got) != want {
        t.Errorf("ReformatFile = %q, want %q", got, want)
    }
    if gotOnDisk := readFile(t, path); gotOnDisk != "(A) task +project\n\nkey:value other\n" {
        t.Errorf("ReformatFile mutated input: %q", gotOnDisk)
    }
}

func TestReformatFilePreservesMissingTrailingNewline(t *testing.T) {
    path := filepath.Join(t.TempDir(), "todo.txt")
    writeFile(t, path, "@desk task +work")
    f := mustTaskFormat(t, defaultTaskFormat)

    got, err := ReformatFile(path, f)
    if err != nil {
        t.Fatal(err)
    }
    if string(got) != "task +work @desk" {
        t.Errorf("ReformatFile = %q, want %q", got, "task +work @desk")
    }
}

func TestReformatFileReturnsReadErrors(t *testing.T) {
    f := mustTaskFormat(t, defaultTaskFormat)
    if _, err := ReformatFile(filepath.Join(t.TempDir(), "missing.txt"), f); err == nil {
        t.Fatal("ReformatFile missing file returned nil error")
    }
}
```

Add `"path/filepath"` to the test imports. The first fixture is intentionally
already in canonical order so it proves blank lines and EOL handling without
introducing unrelated normalization.

- [ ] **Step 2: Run the focused tests and verify the missing API failure.**

Run:

```bash
go test ./internal/todo -run 'TestReformatFile' -count=1
```

Expected: compilation failure because `ReformatFile` does not exist.

- [ ] **Step 3: Refactor line serialization and implement `ReformatFile`.**

Extract the builder in `internal/todo/store.go` into:

```go
func linesData(lines []string, finalNL bool) []byte
```

Make the existing `writeLines` call `os.WriteFile(path, linesData(lines,
finalNL), 0o644)` so all current mutation behavior remains unchanged.

Implement `ReformatFile` in `task_format.go` by calling the existing
`readLines`, applying `f.FormatLine` to every line (including blank lines),
and returning `linesData(formatted, finalNL)`. Do not call `writeLines` or
`os.WriteFile` here.

- [ ] **Step 4: Run the focused tests and all todo tests.**

Run:

```bash
gofmt -w internal/todo/store.go internal/todo/task_format.go internal/todo/task_format_test.go
go test ./internal/todo -run 'TestReformatFile' -count=1
go test ./internal/todo -count=1
```

Expected: focused and full todo suites pass, including existing mutation
tests that pin missing-final-EOL behavior.

- [ ] **Step 5: Commit the file reformatter.**

```bash
git add internal/todo/store.go internal/todo/task_format.go internal/todo/task_format_test.go
git commit -m "feat(todo): preserve file shape during task formatting"
```

## Task 4: Add the `format [FILE]` CLI action

**Files:**

- Create: `internal/cli/format.go`
- Modify: `internal/cli/root.go`
- Create: `internal/cli/testdata/script/t2300-format.txtar`

**Interfaces:**

- Consumes `cfg.TaskFormat` from Task 1 and `todo.ParseTaskFormat`/
  `todo.ReformatFile` from Tasks 2–3.
- Produces a registered Cobra action with `Use: "format [FILE]"`, no alias,
  verbose success lines, and the target/error behavior in the spec.

- [ ] **Step 1: Write the failing CLI session fixture.**

Create `internal/cli/testdata/script/t2300-format.txtar` with this session:

```text
# The default order rewrites both configured files and removes done dates.
exec gtdo format
cmpenv stdout default.stdout
cmp todo.txt default.todo
cmp done.txt default.done

# A custom template and an explicit relative file affect only that file.
exec gtdo -d custom.toml format one.txt
cmpenv stdout custom.stdout
cmp one.txt custom.one
cmp todo.txt default.todo

# Quiet mode suppresses success messages while still rewriting successfully.
env TODOTXT_VERBOSE=0
exec gtdo -d custom.toml format one.txt
stdout '^$'

# Explicit targets must exist and only one target argument is accepted.
! exec gtdo format missing.txt
stderr 'TODO: File .*missing.txt does not exist\.'
! exec gtdo format todo.txt done.txt
stderr '^usage: gtdo format \[FILE\]$'

# Invalid templates fail before any file is rewritten.
! exec gtdo -d invalid.toml format
stderr '^invalid task format: unknown field "unknown"$'
cmp todo.txt default.todo

-- todo.txt --
(B) write report key:one +work @desk 20260808T143045.12Z

plain item
-- done.txt --
x 2026-08-09 finish key:done +archive @desk 20260808T143045.12Z
-- default.stdout --
TODO: $HOME/todo.txt formatted.
TODO: $HOME/done.txt formatted.
-- default.todo --
(B) 20260808T143045.12Z write report key:one +work @desk

plain item
-- default.done --
x 20260808T143045.12Z finish key:done +archive @desk
-- custom.toml --
[paths]
dir = "."

[behavior]
task_format = "[project][content][keywords][context][uuid][priority][checked]"
-- one.txt --
x 2026-08-08 finish key:done +archive @desk 20260808T143045.12Z
-- custom.stdout --
TODO: $HOME/one.txt formatted.
-- custom.one --
+archive finish key:done @desk 20260808T143045.12Z x
-- invalid.toml --
[behavior]
task_format = "[unknown]"
```

The `cmp` commands make the both-file and single-file scope explicit. The
fixture also proves that the date is removed, custom ordering is honored,
quiet mode suppresses only success output, and invalid/missing targets fail.

- [ ] **Step 2: Run the new session and verify the expected unknown-action failure.**

Run:

```bash
go test ./internal/cli -run TestScript -count=1 -v
```

Expected: `t2300-format.txtar` fails before implementation because `format`
is not a registered action; existing scripts may continue to pass.

- [ ] **Step 3: Implement target resolution and the action.**

Create `internal/cli/format.go` with:

```go
func registerFormatAction(root *cobra.Command, cfg *config.Config) {
    root.AddCommand(newAction(actionSpec{
        use:   "format [FILE]",
        short: "Rewrite task files using the configured format.",
        long:  "Rewrites todo.txt and done.txt using the task_format configuration.\nIf FILE is specified, rewrites only that file.",
        run:   actionFormat,
    }, cfg))
}
```

Implement `resolveFormatTargets` with this exact behavior: reject more than
one argument with `usage: gtdo format [FILE]`; no arguments return the two
configured paths in todo-then-done order; one absolute argument is unchanged;
one relative argument is `filepath.Join(cfg.Dir, arg)`. Deduplicate equal
paths so a custom configuration pointing both files at one path is rewritten
and reported once.

Implement `actionFormat` as follows:

1. Validate argument count and parse `cfg.TaskFormat` with
   `todo.ParseTaskFormat`. Print the returned error to stderr and return
   `exitcode.Generic` before creating or writing files when parsing fails.
2. Call `newSession` so the action keeps the existing startup/stream behavior.
3. `os.Stat` each target and reject missing paths or directories with
   `TODO: File <path> does not exist.` and
   `TODO: File <path> is not a regular file.` respectively.
4. Call `todo.ReformatFile` for every target and retain `{path, data}` values
   in memory. If any read fails, return the error before writing any target.
5. Write each retained byte slice with `os.WriteFile(path, data, 0o644)`.
6. If `s.verbose()` is true, print `TODO: <path> formatted.` once per target.

Use `exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)` for errors handled
directly by the action, matching the existing `session.die` convention. Do
not pipe output through the list formatter.

Call `registerFormatAction(root, cfg)` from `NewRootCmd` after the existing
action/list registrations. The Cobra action tree will then automatically
include `format` in help and completion metadata.

- [ ] **Step 4: Run the new fixture and the complete CLI package tests.**

Run:

```bash
gofmt -w internal/cli/format.go internal/cli/root.go
go test ./internal/cli -run TestScript -count=1 -v
go test ./internal/cli -count=1
```

Expected: `t2300-format.txtar` and all existing CLI session/unit tests pass.

- [ ] **Step 5: Commit the action.**

```bash
git add internal/cli/format.go internal/cli/root.go internal/cli/testdata/script/t2300-format.txtar
git commit -m "feat(cli): add task format action"
```

## Task 5: Refresh exact help snapshots and user documentation

**Files:**

- Modify: `internal/cli/testdata/script/help.txtar`
- Modify: `internal/cli/testdata/script/shorthelp.txtar`
- Modify: `internal/cli/testdata/script/defaultaction.txtar`
- Modify: `internal/cli/testdata/script/defaultaction-config.txtar`
- Modify: `internal/cli/testdata/script/t2100-help.txtar`
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Modify: `man/gtdo.1.tmpl`
- Regenerate: `man/gtdo.1`

**Interfaces:**

- Consumes the registered `format` action from Task 4; no new runtime API.
- Produces byte-accurate help snapshots and user-facing documentation for the
  new command/configuration.

- [ ] **Step 1: Run help tests to capture the expected snapshot failures.**

Run:

```bash
go test ./internal/cli -run TestScript -count=1 -v
```

Expected: existing `shorthelp`, `help`, `defaultaction`, and `-vv help`
fixtures fail because the action list now contains `format`.

- [ ] **Step 2: Update all exact action-list snapshots.**

Insert the alphabetically sorted shorthelp line:

```text
    format  Rewrite task files using the configured format.
```

Insert the full help block after `do` and before `help`:

```text
    format [FILE]
      Rewrites todo.txt and done.txt using the task_format configuration.
      If FILE is specified, rewrites only that file.
```

Update every embedded `shorthelp.want`, `help.want`, and `help-vv.want` block
in the five listed fixtures, plus the copied shorthelp blocks in both
`defaultaction` fixtures. Do not change usage, options, environment-variable,
or unrelated action text.

- [ ] **Step 3: Document configuration and usage.**

In `README.md`, add `format` to the in-scope action list and extend the TOML
example:

```toml
[behavior]
verbose = 1
task_format = "[checked][priority][uuid][content][keywords][project][context]"
```

Add a short usage paragraph stating that `gtdo format` rewrites both
configured files and `gtdo format FILE` rewrites one file, and name all seven
placeholders.

Add an `[Unreleased]`/`### Added` changelog bullet in `CHANGELOG.md`:

```text
- Configurable `format` action for normalizing todo.txt and done.txt task fields.
```

In `man/gtdo.1.tmpl`, add a `CONFIGURATION` section documenting
`[behavior] task_format`, its default value, automatic separators, and the
seven fields. Add a sentence to the command/action description that
`format [FILE]` rewrites both configured files or the selected file.

- [ ] **Step 4: Regenerate the committed man page and run documentation tests.**

Run:

```bash
make man
go test ./man ./internal/cli -count=1
```

Expected: `man/gtdo.1` contains the generated `format` command and the new
configuration section; man and CLI tests pass.

- [ ] **Step 5: Commit snapshots and documentation.**

```bash
git add internal/cli/testdata/script/help.txtar internal/cli/testdata/script/shorthelp.txtar internal/cli/testdata/script/defaultaction.txtar internal/cli/testdata/script/defaultaction-config.txtar internal/cli/testdata/script/t2100-help.txtar README.md CHANGELOG.md man/gtdo.1.tmpl man/gtdo.1
git commit -m "docs: document configurable task formatting"
```

## Task 6: Run the full verification gate

**Files:**

- Verify: all repository files changed by Tasks 1–5.

**Interfaces:**

- Consumes the complete implementation and documentation commits.
- Produces verified build/test/lint/man outputs before the feature is called
  complete.

- [ ] **Step 1: Run the complete Go test suite.**

```bash
go test ./...
```

Expected: every package, unit test, testscript session, and man test passes.

- [ ] **Step 2: Run the production build.**

```bash
make build
```

Expected: `./gtdo` builds successfully with the repository's version
metadata.

- [ ] **Step 3: Run formatting, static checks, and generated-man verification.**

```bash
gofmt -w internal/config/defaults.go internal/config/toml.go internal/config/config.go internal/config/config_test.go internal/todo/store.go internal/todo/task_format.go internal/todo/task_format_test.go internal/cli/format.go internal/cli/root.go
git diff --check
make lint
make man
go test ./...
```

Expected: gofmt produces no further diff, lint is clean, regeneration is
stable, and the final full suite remains green.

- [ ] **Step 4: Inspect the final diff and status.**

```bash
git status --short
git diff HEAD~5..HEAD --stat
git log --oneline -6
```

Confirm that only the approved formatter implementation, tests, snapshots,
documentation, generated man page, and the already committed spec/plan are
present; no build binary or unrelated file is staged.

- [ ] **Step 5: Commit any final mechanical formatting only if required.**

If Step 3 changes tracked Go/man files after the feature commits, review the
diff and commit only those intentional generated/formatting changes:

```bash
git add internal man/gtdo.1
git commit -m "style: format task formatter changes"
```

If there is no diff, make no empty commit.

## Plan Self-Review

- **Spec coverage:** Configuration default/TOML resolution is Task 1; all seven
  fields, separators, checked-date removal, UUID/keyword rules, and custom
  order are Task 2; blank lines and trailing EOL are Task 3; both-file/single-
  file scope, preflight reads, regular-file checks, quiet/success output, and
  argument validation are Task 4; help, README, changelog, and generated-man
  documentation are Task 5; verification is Task 6.
- **Placeholder scan:** The plan contains no `TBD`, `TODO`, “implement later”,
  “similar to Task”, or unspecified “add appropriate handling” steps. Every
  production interface, test command, error prefix, and commit command is
  named explicitly.
- **Type consistency:** Task 1 produces `Config.TaskFormat`; Task 2 defines
  `todo.TaskFormat`, `ParseTaskFormat`, and `FormatLine`; Task 3 defines
  `ReformatFile`; Task 4 consumes those exact names and returns the CLI's
  existing `exitcode` errors. No later task references an alternate function
  name or field type.
