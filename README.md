# gtdo

gtdo is a Go port of [todo.txt-cli](https://github.com/todotxt/todo.txt-cli)'s
`todo.sh`: the classic command-line interface for the
[todo.txt](http://todotxt.org/) format, as a single static binary.

## Parity

gtdo's goal is parity with todo.sh (todo.txt-cli v2.x): for the twenty
in-scope actions it produces byte-identical stdout, stderr, exit codes, and
file states. The in-scope actions are:

`add`, `addm`, `addto`, `append`, `archive`, `del`, `depri`, `do`, `help`,
`list`, `listall`, `listcon`, `listpri`, `listproj`, `move`, `prepend`, `pri`,
`replace`, `report`, `shorthelp`

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
gtdo [-fhpantvV] [-d todo_config] action [task_number] [task_description]
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

gtdo reads `~/.config/gtdo/config.toml` (override with `-d PATH` or
`$GTDO_CONFIG`). A missing file is not an error — every value has a default:

```toml
[paths]
dir = "~/todo"          # TODO_DIR

[behavior]
verbose = 1             # TODOTXT_VERBOSE
force = false           # TODOTXT_FORCE
```

The usual todo.txt environment variables (`TODO_DIR`, `TODO_FILE`,
`DONE_FILE`, `REPORT_FILE`, `TODOTXT_*`) keep working for scripting
compatibility.

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
