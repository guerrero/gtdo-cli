# Allowed Contexts and Projects Design

## Goal

Add two TOML configuration options that allow a user to define the only
contexts and projects accepted in task text. Any other context or project in a
new or edited task must fail with an error that identifies the configured file
and the task's 1-based line number.

## Configuration contract

The options live in `[behavior]` and are arrays of complete, sigil-prefixed
tokens:

```toml
[behavior]
allowed_contexts = ["@work", "@home"]
allowed_projects = ["+gtdo", "+personal"]
```

Matching is exact and case-sensitive. A missing option means that category is
unrestricted for backward compatibility. An explicitly empty array means that
no token of that category is permitted. The options are TOML-only; no new
environment-variable forms are added.

`config.Config` exposes the resolved slices. The config package remains
independent of task parsing; the CLI constructs a `todo.SigilPolicy` from the
resolved values when it creates a session.

## Validation boundary and data flow

`todo.Store` carries the policy and enforces it for operations that accept task
text:

- `add`, `addm`, and `addto` validate after cleaning input and applying the
  date-on-add and priority-on-add transformations.
- `append`, `prepend`, and `replace` validate the fully rebuilt candidate line,
  including any preserved priority/date prefix.

Operations that only manipulate existing task metadata or move existing text
(`pri`, `depri`, `do`, `del`, `move`, `archive`, and `report`) do not introduce
new sigils and do not newly reject legacy tasks that were already on disk.
Read-only listing remains available even when existing tasks contain tags that
are no longer allowed.

The validator walks whitespace-delimited task words from left to right and
returns the first disallowed context or project. It reuses the same sigil-word
classification as task parsing, so punctuation and embedded sigils retain the
existing behavior.

`addm` precomputes all final lines, assigns their eventual destination line
numbers, validates every line, and writes only after the complete batch passes.
This prevents a later invalid line from leaving earlier lines in the file.
All other guarded mutations validate before their write operation as well.

## Errors

Validation returns a structured error containing the token kind, token text,
destination path, and 1-based line number. The CLI prints it through the
existing `die` path and exits with status 1; no success confirmation is
printed, and the target file remains unchanged.

The rendered messages are:

```text
TODO: Context "@bad" is not allowed in /path/to/todo.txt at line 2.
TODO: Project "+bad" is not allowed in /path/to/todo.txt at line 2.
```

For `add` and `addto`, the line is the next line in the destination file. For
`addm`, each candidate uses its eventual line number. For text edits, the
existing task line number is reported. The actual configured path is rendered,
not the short `TODO`/`DONE` prefix.

## Files and tests

Configuration schema and resolution changes belong in `internal/config`:

- extend the TOML behavior schema and resolved `Config` with the two slices;
- cover missing, populated, and explicitly empty arrays in config tests.

Policy and mutation enforcement belong in `internal/todo`:

- add a policy and structured validation error;
- attach the policy to `Store`;
- validate all guarded mutation candidates before writes;
- add unit tests for exact matching, unrestricted mode, error details, and
  atomic multiline adds.

Session wiring and the end-to-end contract belong in `internal/cli`:

- pass the resolved policy through `newSession`;
- add a testscript that exercises the TOML options and pins stderr, exit code,
  stdout silence, and unchanged files.

User-facing documentation belongs in `README.md` and `man/gtdo.1.tmpl`; the
generated `man/gtdo.1` is refreshed with `make man`.

Verification runs focused package tests, `go test ./...`, lint, and the
man-page consistency test.
