# gtdo

gtdo is a Go port of [todo.txt-cli](https://github.com/todotxt/todo.txt-cli)'s
`todo.sh`: the classic command-line interface for the
[todo.txt](http://todotxt.org/) format, as a single static binary.

## Parity

gtdo's goal is parity with todo.sh (todo.txt-cli v2.x): for the twenty-one
in-scope actions it produces byte-identical stdout, stderr, exit codes, and
file states. The in-scope actions are:

`add`, `addm`, `addto`, `append`, `archive`, `del`, `depri`, `do`, `format`, `help`,
`list`, `listall`, `listcon`, `listpri`, `listproj`, `move`, `prepend`, `pri`,
`replace`, `report`, `shorthelp`

Addons, `command`, `deduplicate`, and `listfile` are out of scope. The
migration checklist lives in [ACTIONS.md](ACTIONS.md).

## Quickstart

```bash
gtdo add "water the plants"
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

## Configuration

gtdo reads `~/.config/gtdo/config.toml` (override with `-d PATH` or
`$GTDO_CONFIG`). A missing file is not an error — every value has a default:

```toml
[paths]
dir = "~/todo"          # TODO_DIR

[behavior]
verbose = 1             # TODOTXT_VERBOSE
force = false           # TODOTXT_FORCE
task_format = "[checked][priority][uuid][content][keywords][project][context]"
```

`gtdo format` rewrites both configured task files, while `gtdo format FILE`
rewrites only the selected file. The `task_format` template supports
`[checked]`, `[priority]`, `[uuid]`, `[content]`, `[keywords]`, `[project]`,
and `[context]` placeholders.

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
