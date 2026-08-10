# gtdo

gtdo is a Go port of [todo.txt-cli](https://github.com/todotxt/todo.txt-cli)'s
`todo.sh`: the classic command-line interface for the
[todo.txt](http://todotxt.org/) format, as a single static binary.

## Parity

gtdo's goal is parity with todo.sh (todo.txt-cli v2.x): for the twenty
in-scope upstream actions it produces byte-identical stdout, stderr, exit
codes, and file states. The in-scope upstream actions are:

`add`, `addm`, `addto`, `append`, `archive`, `del`, `depri`, `do`, `help`,
`list`, `listall`, `listcon`, `listpri`, `listproj`, `move`, `prepend`, `pri`,
`replace`, `report`, `shorthelp`

The `format` action is an additional gtdo extension rather than an upstream
todo.sh parity action.

Addons, `command`, `deduplicate`, and `listfile` are out of scope. The
migration checklist lives in [ACTIONS.md](ACTIONS.md).

## Quickstart

```bash
gtdo add "water the plants"
gtdo add -i
gtdo add -g --only context
gtdo list
gtdo do 1
```

## Install

Build from source:

```bash
make build    # builds ./gtdo with version metadata
make install  # installs gtdo into $GOBIN
```

or with `go install`:

```bash
go install github.com/guerrero/gtdo/cmd/gtdo@latest
```

## Usage

```
gtdo [-fhpanvV] [-d todo_config] action [task_number] [task_description]
```

Flags go before the action, getopts-style, exactly like todo.sh. `gtdo -h`
lists the actions, `gtdo -V` prints the version block.

### Interactive add

Use `gtdo add -i` (or `--interactive`) for one editable task line with
`@context` and `+project` completion. Use `gtdo add -g` (or `--guided`) for
phased input: metadata, projects, then contexts. The phases are named
`metadata`, `project`, and `context`; select a subset with repeatable
`--only` options:

```bash
gtdo add -i
gtdo add -g
gtdo add --guided --only metadata
gtdo add --guided --only project --only context
```

Candidates are deduplicated from the configured `todo_file` and `done_file`
only; `report_file` is never scanned. On a terminal, project and context
phases use a searchable selector: Up/Down moves, Space toggles, `/` filters,
Enter confirms, and Esc clears the filter or exits. With non-TTY stdin, no
escape sequences or prompts are emitted. Interactive mode reads one task line;
guided mode reads one task line, then input for each enabled phase in order:
zero or more `key:value` metadata lines ending with an empty line, one
space-separated project line, and one space-separated context line. Phases
omitted with `--only` consume no line. Empty selection lines are valid.

## Configuration

gtdo searches `-d PATH`, `$GTDO_CONFIG`, `~/.config/gtdo/config.json`, then
`/etc/gtdo/config.json`. JSON is a hard switch: gtdo only reads JSON
configuration. A missing file is not an error — every value has a default:

```json
{
  "dir": "~/todo",
  "files": {
    "todo": "~/todo/todo.txt",
    "done": "~/todo/done.txt",
    "report": "~/todo/report.txt"
  },
  "behaviour": {
    "verbose": 1,
    "force": false,
    "preserveLineNumbers": true,
    "enableUUID": false,
    "taskFormat": "[checked][priority][uuid][content][keywords][project][context]",
    "allowedContexts": ["@work", "@home"],
    "allowedProjects": ["+gtdo", "+personal"]
  }
}
```

Unknown keys and `null` are errors; omitted settings retain their defaults.
`gtdo format` rewrites both configured task files, while `gtdo format FILE`
rewrites only the selected file. The `taskFormat` template supports
`[checked]`, `[priority]`, `[uuid]`, `[content]`, `[keywords]`, `[project]`,
and `[context]` placeholders.

`allowedContexts` and `allowedProjects` are JSON allow-lists for context and
project tags. Omit either list to leave its category unrestricted; an explicit
empty list rejects every tag in that category. Matching is exact and
case-sensitive. gtdo validates final task text for `add`, `addm`, `addto`,
`append`, `prepend`, and `replace`; `list` remains usable for legacy tasks
with tags no longer allowed. A rejected tag reports its category, token, file,
and 1-based line number as `TODO: Context "@home" is not allowed in
/path/to/todo.txt at line 3.` (with `Project` for a project tag).

The usual todo.txt environment variables (`TODO_DIR`, `TODO_FILE`, `DONE_FILE`,
`REPORT_FILE`, `TODOTXT_FORCE`, `TODOTXT_PRESERVE_LINE_NUMBERS`,
`TODOTXT_AUTO_ARCHIVE`, `TODOTXT_PRIORITY_ON_ADD`, `TODOTXT_VERBOSE`,
`TODOTXT_DEFAULT_ACTION`, `TODOTXT_SOURCEVAR`, `TODOTXT_PLAIN`,
`GTDO_ENABLE_UUID`, and `SENTENCE_DELIMITERS`) keep working for scripting
compatibility.

Timestamp IDs are opt in. Add this to the `behaviour` section to assign an ID to
each newly created task:

```json
{"behaviour":{"enableUUID":true}}
```

`GTDO_ENABLE_UUID` overrides the JSON value. When enabled, gtdo adds a UTC
timestamp ID in the exact `YYYYMMDDTHHMMSS.nnZ` format (for example,
`20260808T143045.12Z`). If a candidate collides with an existing ID, or with
one allocated earlier in the same batch, gtdo advances it by 10 ms and retries.
IDs are creation-only and preserved through edits and moves; enabling the
setting later never backfills existing tasks.

The former date-on-add behavior has been retired, so add commands no longer
insert a creation date automatically. Dates already present in a task remain
unchanged.

## Development

```bash
make build   # build ./gtdo
make test    # go test ./...
make lint    # golangci-lint
make man     # regenerate man/gtdo.1
make release # publish a tagged release via goreleaser
```

## License

MIT — see [LICENSE](LICENSE).
