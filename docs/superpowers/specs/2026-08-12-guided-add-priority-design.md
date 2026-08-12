# Guided Add Priority Phase Design

## Goal

Add a priority step to `gtdo add -g` (guided mode) so the user can set a
priority or leave it empty, and reorder the guided phases. The new phase
order is:

```text
Task → priority → contexts → projects → metadata
```

The priority phase runs by default and is selectable through `--only
priority`. An empty priority leaves the task without a guided priority, so
the existing `priority_on_add` configuration applies exactly as it does
today. This design supersedes the phase ordering and `--only` list in
`2026-08-10-interactive-add-design.md`, which stays as history.

## Command contract

`priority` becomes a fourth guided phase. `--only` accepts it like the
others:

```text
gtdo add -g                       # all four phases, in the new order
gtdo add -g --only priority       # only the priority phase
gtdo add -g --only context --only priority
```

Without `--only`, guided mode runs all phases in the order task, priority,
contexts, projects, metadata. The task prompt always runs, as today.

Usage and help listings follow the new order:

- `addUsage()` becomes
  `usage: gtdo add [-i|--interactive|-g|--guided] [--only priority|context|project|metadata]`.
- The add long help line becomes:
  "Interactive modes: -i|--interactive adds one editable task; -g|--guided
  runs priority, context, project, and metadata phases." followed by
  "Guided mode accepts repeatable --only priority|context|project|metadata."
  — pinned byte-for-byte in help.txtar.

## Skip rule

When the base task text already carries a priority (detected with
`todo.Task.Priority()`), the priority phase is skipped entirely: no prompt
on a terminal, no line consumed from a pipe. The task keeps its own
priority and no second `(X) ` prefix is ever composed. This mirrors the
existing rule that sigils already present in the task are not appended a
second time, and it composes cleanly with `--only priority` (a skipped
phase is a no-op).

## Input and validation

Both input adapters accept exactly two forms:

- an empty line: skip the phase;
- a single ASCII letter `a`–`z` or `A`–`Z`: the priority.

Anything else (for example `high`, `(A)`, or `AB`) returns an error in the
shape of the existing metadata errors, e.g.
`invalid priority "high": expected a single letter A-Z`. The error cancels
the whole add without writing, matching how the metadata phase rejects
malformed `key:value` lines.

### Terminal input

`ttyAddInput.PromptPriority` opens the shared readline instance with the
prompt `Priority (A-Z, empty to skip): ` and validates the returned line.

### Pipe protocol

`lineAddInput.PromptPriority` consumes exactly one line. The deterministic
line protocol becomes:

```text
<task line>
<priority line, empty to skip>
<context line, space-separated selections>
<project line, space-separated selections>
<metadata key:value lines, terminated by an empty line>
```

As before, the protocol consumes exactly the lines assigned to the enabled
phases; a skipped priority phase consumes no line.

## Composition

`composeGuidedTask` gains a priority parameter and takes the remaining
groups in the new phase order:
`composeGuidedTask(base, priority, contexts, projects, metadata)`.

A non-empty priority is uppercased and prepended to the task with the
existing `appendGuidedToken` join, so `a` becomes `(A) `:

```text
Call team +gtdo @home due:tomorrow   with priority "a"
→ (A) Call team +gtdo @home due:tomorrow
```

An empty priority composes nothing.

The store path is unchanged: `add.go` continues to pass
`cfg.PriorityOnAdd` to `Store.Add`. `prepareAdd` already uppercases a
leading lowercase priority and already suppresses the config priority when
the text carries one (`priorityOnAddRe`), so a guided priority wins over
the config, and an empty guided priority lets the config apply exactly as
in a regular `gtdo add`.

## Architecture

No new packages and no store changes. The delta is confined to:

1. `internal/cli/add_options.go` — `phasePriority` constant,
   `parseGuidedPhase`, `isGuidedPhase`, and `addUsage()`.
2. `internal/cli/add_guided.go` — `PromptPriority` on the `addInput`
   interface, the new `runGuided` phase order with the skip rule, the
   priority-aware `composeGuidedTask`, the `lineAddInput` priority line,
   and the shared priority validator.
3. `internal/cli/add_terminal.go` — `ttyAddInput.PromptPriority`.
4. `internal/cli/actions.go` — add long-help text.

## Testing

Unit tests cover:

- `parseGuidedPhase("priority")` and the `--only priority` combination;
- `phaseEnabled` defaults with the new phase set;
- `runGuided`: the task → priority → context → project → metadata call
  order, the skip rule when the base carries a priority (including with
  `--only priority`), and composition with and without a priority;
- `composeGuidedTask`: prepend position, empty priority, lowercase
  uppercasing, empty base;
- the pipe protocol: priority line placement and consumption, empty-line
  skip, and invalid-priority errors (including `invalid priority %q` text);
- the `addInput` fake records the priority prompt in the call sequence.

CLI testscript updates:

- `t2300-interactive-add.txtar`: pipe protocol gains the priority line in
  every guided scenario, composed output gains `(A) ` where set, the usage
  line gains `priority`, plus new scenarios for `--only priority` and a
  base task that already carries a priority;
- `help.txtar`: the two guided-mode help lines.

## Compatibility

The default `add` action, `addm`, `addto`, interactive mode, all global
flags, and the store are unchanged. The only observable changes to
existing behavior are the guided phase order and the `--only` list — both
intentional and pinned in the txtar suite.
