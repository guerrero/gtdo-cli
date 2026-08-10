# Task Format Design

## Goal

Add a `gtdo format` action that rewrites todo.txt task lines into a
configurable field order. With no file argument it rewrites the configured
`todo.txt` and `done.txt`; with one file argument it rewrites only that file.

## User-facing behavior

The default configuration is:

```json
{
  "behaviour": {
    "taskFormat": "[checked][priority][uuid][content][keywords][project][context]"
  }
}
```

The formatter recognizes the fields `[checked]`, `[priority]`, `[uuid]`,
`[content]`, `[keywords]`, `[project]`, and `[context]`. Their order in the
template controls the output order. The formatter inserts one ASCII space
between adjacent non-empty fields, regardless of whether the template has
spaces between its placeholders. Empty fields are omitted, so output never
has leading, trailing, or doubled separator spaces caused by absent fields.

Field extraction follows the existing todo.txt conventions:

- `checked` is `x` for a line beginning with the completed marker and is empty
  otherwise. An optional completion date immediately following `x` is
  consumed and not emitted.
- `priority` is the leading `(A)`-style priority, including its parentheses.
- `uuid` is an ISO-like value matching `YYYYMMDDTHHMMSS.ssZ`.
- `keywords` are whitespace-delimited `key:value` words.
- `project` and `context` are the existing `+name` and `@name` sigil words.
- `content` is every remaining word, in source order.

Repeated values within a field retain source order and are separated by one
space. Blank lines remain blank. The source file's trailing-newline state is
preserved.

The formatter consumes the current `x YYYY-MM-DD` done marker date. The date
is not represented in the output because the canonical format has only the
`checked` field; completed output therefore starts with `x` when that field is
present.

## Command and file handling

`gtdo format [FILE]` is a mutating action. Without `FILE`, the action targets
the configured todo and done files. With one `FILE`, it targets only that
file; relative paths are resolved under the configured TODO directory and
absolute paths are accepted.

The template is parsed and validated before any selected file is changed.
All selected files are read and formatted before writing, preventing a
malformed task from causing a later selected file to be skipped after a
partial read phase. Blank lines and newline shape are handled by the existing
line store. An explicit file must be a regular existing file.

On successful formatting, verbose mode prints one line per target:

```text
TODO: <file> formatted.
```

With `TODOTXT_VERBOSE=0`, the action prints no success lines. Invalid template
fields, malformed templates, missing explicit files, and non-regular targets
fail without rewriting files.

## Configuration

Add `TaskFormat` to the resolved config and `taskFormat` to the `behaviour`
JSON schema. The built-in default is the canonical format above. This setting
is intentionally configuration-only; it has no environment-variable or
global-flag override.

## Internal design

`internal/todo` owns a pure template parser and line formatter. Parsing a
template yields a sequence of the seven known field identifiers and rejects
unknown or malformed bracket fields. Formatting a line extracts the fields,
renders each requested non-empty value, and joins the values with one space.
The CLI action resolves targets and delegates file reads/writes to the store.

The formatter does not alter existing list, filter, sort, or mutation
behavior. Existing parsing helpers are reused where their semantics match;
the UUID matcher and checked-date consumption are formatter-specific.

## Testing and documentation

- Unit tests cover the default order, custom order, omitted fields, repeated
  fields, UUID and keyword extraction, completion-date removal, malformed
  templates, blank lines, and trailing-newline preservation.
- Config tests cover the default and JSON-loaded `taskFormat` value.
- CLI session tests cover both-file formatting, single-file formatting,
  explicit-file failures, quiet mode, and success messages.
- Help text, README/config examples, the changelog, and the generated man page
  document the new action and setting.
