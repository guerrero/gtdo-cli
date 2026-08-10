# Interactive Add Modes Design

## Goal

Add two explicit interactive modes to `gtdo add` without changing the
existing positional-argument behavior:

```text
gtdo add -i              # one-line editable input with @/+ completion
gtdo add --interactive
gtdo add -g              # phased guided input
gtdo add --guided --only context
```

The modes must create the same task text and use the same store mutation,
configuration, numbering, output, and validation semantics as a regular
`gtdo add`.

## Command contract

`-i` is the shorthand for `--interactive`; `-g` is the shorthand for
`--guided`. The two modes are mutually exclusive. `--only` is repeatable,
requires guided mode, and accepts exactly `metadata`, `project`, or
`context`:

```text
gtdo add --guided --only metadata
gtdo add --guided --only project --only context
```

When `--only` is absent, guided mode runs all three optional phases. The task
prompt always runs, including when `--only` is supplied. Existing invocations
such as `gtdo add "Buy milk"` and `gtdo add` retain their current behavior.

The mode options are parsed as action-local options after `add`; existing
global getopts-style flags continue to be parsed before the action. Guided
mode does not accept positional task text, and interactive mode does not
accept positional task text; invalid combinations use the existing usage
error path and write nothing. Because `-f` explicitly suppresses interactive
input for legacy actions, combining `-f` with either new mode is also a usage
error rather than a silent no-op.

## Candidate sources

Completion and selection data are collected from the two configured
task-bearing paths: `Config.TodoFile` and `Config.DoneFile`. `ReportFile` is
not scanned. Each existing readable file is read best-effort; a missing or
unreadable file contributes no candidates. The collector itself never creates
candidate files; normal add startup may still ensure the configured files as
it does for every existing add action.

The collector extracts and deduplicates:

- `@context` and `+project` sigil words using the existing `todo.SigilWords`
  classification rules;
- `key:value` metadata tokens, preserving the complete key and value text.

Contexts, projects, metadata keys, and metadata values are sorted with a
stable byte-wise order before being shown. Tokens already present in the
entered task are not added a second time by guided selection.

## Interactive mode (`add -i`)

On a terminal, the mode opens an editable `Add: ` line using a terminal
line-editor library. The editor supports normal cursor editing and dynamic
completion for the current word:

```text
Add: Call Alice @<Tab>
@home  @office  @phone
```

Typing `@` filters contexts and typing `+` filters projects. Tab accepts or
cycles matching candidates according to the line editor's standard behavior;
the selected token remains editable. Enter submits the complete line to the
existing `Store.Add` pipeline.

When stdin is not a terminal, interactive mode falls back to reading one
plain line using the existing non-TTY prompt behavior. No terminal escape
sequences are written, so pipes and testscript sessions remain deterministic;
completion is unavailable in this fallback.

## Guided mode (`add -g`)

Guided mode first asks for the task text with an editable line. It then runs
the selected optional phases in this order: metadata, projects, contexts.

### Metadata phase

The phase repeatedly asks for a metadata key. Existing keys are offered as
completion candidates, and a custom key can be entered. An empty key ends the
phase. For a key, existing values observed for that key are offered as
completion candidates; a custom value can be entered. An empty value skips
that entry. Each accepted pair is appended as `key:value`, and the same key
may be selected again for another value.

### Project and context phases

Each phase displays a searchable multi-select list of the candidates for its
sigil category and shows this help line:

```text
↑/↓ move  Space toggle  / filter  Enter confirm  Esc clear/exit
```

Up/Down moves the cursor, Space toggles the highlighted item, `/` starts a
filter, Enter confirms the current selection, and Esc clears an active filter
or exits the selector with no new selection. Empty selections are valid.
Selected tokens are appended to the task in deterministic sorted order and
are not duplicated when the entered task already contains the token.

When stdin is not a terminal, guided mode uses line-oriented prompts without
escape sequences. The task is one line, metadata is supplied as successive
`key:value` lines terminated by an empty key line, and project/context phases
accept a space-separated line of selected tokens. This makes the feature
scriptable while preserving the same final task composition.

## Mutation and cancellation

After all requested phases complete, the assembled text is passed once to the
existing `Store.Add` operation with the resolved date-on-add and
priority-on-add settings. No file is changed until all input is complete.
The regular verbose `N text` and `TODO: N added.` output is unchanged.

EOF or Ctrl-C before submission cancels the operation without writing a task.
Invalid mode flags, unknown `--only` values, and unsupported positional
arguments return the existing usage error and leave the destination file
unchanged. Candidate-file read errors are intentionally non-fatal.

## Architecture

The CLI gains an action-local add-options parser and a small interactive input
package with three boundaries:

1. **Candidate collector** reads the configured task-bearing files and returns
   sorted contexts, projects, metadata keys, and per-key values. It has no
   terminal or mutation dependencies and is unit-testable with temporary
   files.
2. **Input adapters** expose one line-editor adapter for TTY input and one
   deterministic line adapter for non-TTY input. The TTY adapter uses
   `github.com/chzyer/readline` for cursor editing and dynamic completion;
   guided selectors render on top of the same terminal input boundary.
3. **Add flow** validates mode options, obtains the task and optional
   selections, composes the final text, and delegates exactly once to the
   existing session/store add path.

The candidate collector is shared by inline completion and guided prompts so
both modes always see the same values. No changes are made to the existing
shell completion command beyond exposing the new action-local options in
action help where Cobra can do so without taking over global flag parsing.

## Testing

Unit tests cover:

- parsing `-i`, `--interactive`, `-g`, `--guided`, repeated `--only`, mutual
  exclusion, and invalid phase names;
- union, deduplication, sorting, missing-file handling, and metadata grouping
  across `todo_file` and `done_file`;
- guided text composition, duplicate-token suppression, empty selections, and
  the non-TTY line protocol;
- the inline completer's handling of `@` and `+` prefixes and typed-prefix
  filtering.

CLI testscript sessions pin stdout, stderr, exit status, and todo-file bytes
for scripted interactive and guided adds. The full existing `go test ./...`
suite remains the compatibility gate.

## Compatibility and scope

The default `add` action, `addm`, `addto`, all existing global flags, and all
existing output strings remain unchanged. The feature adds no environment
variables, no new task syntax, and no report-file scanning. The only new
runtime dependency is the line-editor package required for TTY editing.
