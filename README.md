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
allowed_contexts = ["@work", "@home"]
allowed_projects = ["+gtdo", "+personal"]
```

`gtdo format` rewrites both configured task files, while `gtdo format FILE`
rewrites only the selected file. The `task_format` template supports
`[checked]`, `[priority]`, `[uuid]`, `[content]`, `[keywords]`, `[project]`,
and `[context]` placeholders.

`allowed_contexts` and `allowed_projects` are TOML-only allow-lists for
context and project tags. Omit either list to leave its category unrestricted;
an explicit empty list rejects every tag in that category. Matching is exact
and case-sensitive. gtdo validates final task text for `add`, `addm`, `addto`,
`append`, `prepend`, and `replace`; `list` remains usable for legacy tasks
with tags no longer allowed. A rejected tag reports its category, token, file,
and 1-based line number as `TODO: Context "@home" is not allowed in
/path/to/todo.txt at line 3.` (with `Project` for a project tag).

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
