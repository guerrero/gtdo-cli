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

The usual todo.txt environment variables (`TODO_DIR`, `TODO_FILE`, `DONE_FILE`,
`REPORT_FILE`, `TODOTXT_FORCE`, `TODOTXT_PRESERVE_LINE_NUMBERS`,
`TODOTXT_AUTO_ARCHIVE`, `TODOTXT_PRIORITY_ON_ADD`, `TODOTXT_VERBOSE`,
`TODOTXT_DEFAULT_ACTION`, `TODOTXT_SOURCEVAR`, `TODOTXT_PLAIN`, and
`SENTENCE_DELIMITERS`) keep working for scripting compatibility.

Timestamp IDs are opt in. Add this to the behavior section to assign an ID to
each newly created task:

```toml
[behavior]
enable_uuid = true        # add UTC timestamp IDs to new tasks
```

`GTDO_ENABLE_UUID` overrides the TOML value. When enabled, gtdo adds a UTC
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
