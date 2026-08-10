# JSON Configuration Design

## Goal

Replace gtdo's TOML configuration with a JSON-only format. This is a hard
switch: gtdo will not discover, parse, or provide compatibility aliases for
the previous TOML format.

The change affects only configuration representation and discovery. Existing
runtime defaults, CLI and environment precedence, path expansion, color
resolution, and todo.sh parity remain unchanged.

## Configuration discovery

gtdo searches for the first existing regular file in this order:

1. The path passed with `-d PATH`.
2. `$GTDO_CONFIG`.
3. `~/.config/gtdo/config.json`.
4. `/etc/gtdo/config.json`.

Explicit `-d` and `$GTDO_CONFIG` paths may have any filename, but their
contents must be valid JSON. A missing candidate falls through to the next
location. If no candidate exists, gtdo uses its defaults. The legacy default
locations ending in `config.toml` are not searched.

## JSON schema

The document has four top-level properties: `dir`, `files`, `behaviour`, and
`colors`. Compound property names use camelCase. `behaviour` deliberately uses
British spelling; `colors` retains its existing American spelling.

```json
{
  "dir": "~/todo",
  "files": {
    "todo": "~/todo/todo.txt",
    "done": "~/todo/done.txt",
    "report": "~/todo/report.txt"
  },
  "behaviour": {
    "force": false,
    "preserveLineNumbers": true,
    "autoArchive": true,
    "dateOnAdd": false,
    "priorityOnAdd": "",
    "verbose": 1,
    "defaultAction": "",
    "sourceVar": "",
    "sentenceDelimiters": ",.:;"
  },
  "colors": {
    "priA": "yellow",
    "priB": "green",
    "priC": "light_blue",
    "priX": "white",
    "colorDone": "light_grey",
    "colorProject": "",
    "colorContext": "",
    "colorDate": "",
    "colorNumber": "",
    "colorMeta": "",
    "map": {
      "yellow": "\\033[1;33m"
    }
  }
}
```

`colors` supports `priA` through `priZ`, the six `color*` roles shown above,
and `map`. Color values retain their current meaning: a built-in or overridden
map name, a raw ANSI string, or an empty string to disable the role.

All properties are optional. Omitted values retain the current defaults.
When a file path is omitted, it is derived from the fully resolved `dir` as it
is today. JSON `null` is not a supported substitute for a setting value; each
present property must have the type shown by the schema.

## Architecture and data flow

`internal/config` uses Go's standard `encoding/json` package and dedicated
typed schema structs. The TOML decoder dependency is removed. The public
runtime `Config` type remains unchanged so configuration consumers in the CLI
and UI do not acquire format-specific knowledge.

Loading proceeds in this order:

1. Initialize the file schema with todo.sh-compatible defaults.
2. Strictly decode the selected JSON document over those defaults.
3. Apply environment-variable overrides.
4. Apply CLI-flag overrides.
5. Expand leading `~` and `$HOME` in paths and derive omitted file paths from
   the resolved `dir`.
6. Resolve color names and raw ANSI values using the existing color logic.

The effective precedence remains CLI flags over environment variables over
the JSON file over defaults. Existing special handling for `-v` and
`TODOTXT_VERBOSE` is unchanged.

## Validation and errors

Decoding is strict. gtdo rejects:

- malformed JSON;
- multiple or trailing JSON values;
- unknown properties at any nesting level;
- values whose JSON type does not match the schema;
- `null` for a setting value; and
- incorrect nesting.

Read failures retain the existing `config: read <path>: ...` context. Decode
and validation failures use `config: parse <path>: ...`. Configuration errors
remain fatal instead of silently falling back to defaults.

## Documentation and dependency changes

The implementation updates all live and normative documentation that names
the format or default paths: README, AGENTS.md, man-page template and generated
page, package comments, and the original migration design and plans. Examples
use the new JSON schema and terminology. The BurntSushi TOML dependency is
removed from `go.mod` and `go.sum`.

Historical release entries are not rewritten unless they claim current TOML
support. No migration command or automatic converter is added.

## Testing

Config unit tests use JSON fixtures and cover:

- the full camelCase schema;
- default, JSON, environment, and CLI precedence;
- JSON search order and path expansion;
- legacy `config.toml` default locations being ignored;
- unknown properties at the top level and within nested objects;
- malformed JSON, trailing values, type mismatches, `null`, and incorrect
  nesting;
- derived file paths;
- named, overridden, raw, disabled, and fallback colors; and
- missing configuration yielding defaults.

Affected CLI txtar fixtures are converted to JSON, including default-action
configuration. Verification includes formatting, the full Go test suite,
`go vet ./internal/config/`, man-page regeneration checks, and a
repository-wide scan for stale TOML code and current-documentation references.

## Out of scope

- Reading TOML files or legacy snake_case keys.
- Supporting both formats during a transition period.
- Automatically converting user configuration.
- Changing CLI flags, environment variable names, defaults, or runtime
  behavior unrelated to configuration parsing.
