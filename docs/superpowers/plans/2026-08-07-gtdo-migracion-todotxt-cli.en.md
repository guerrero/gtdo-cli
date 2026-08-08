# Design — gtdo: port of todo.txt-cli to Go

- **Date**: 2026-08-07
- **Status**: approved
- **Reference**: [todo.txt-cli](https://github.com/todotxt/todo.txt-cli) (todo.sh v2.x)
- **Structure reference**: [gitia](https://github.com/guerrero/gitia) (local repository)

## 1. Context and goals

Port the todo.txt CLI (todo.sh, ~1571 lines of bash) to a Go binary named **`gtdo`** with exactly the same behavior. The reference for file structure, tooling and conventions is the local `gitia` repo (cobra, `internal/*` packages, txtar tests with go-internal, Makefile, goreleaser, man pages).

The central acceptance criterion: **for a given input, the output (stdout, stderr, exit code, and file state) must be byte-identical to todo.sh**, except for the exclusions declared in §2 and gtdo's own help/version texts (§6.4).

## 2. Scope

### In scope

- **20 actions** with their aliases: `add` (a), `addm`, `addto`, `append` (app), `archive`, `del` (rm), `depri` (dp), `do` (done), `help`, `shorthelp`, `list` (ls), `listall` (lsa), `listcon` (lsc), `listpri` (lsp), `listproj` (lsprj), `move` (mv), `prepend` (prep), `pri` (p), `replace`, `report`.
- Global flags: `-@ -+ -a -A -c -d -f -h -n -N -p -P -t -T -v -V -x` (semantics identical to todo.sh, §6.1).
- Own TOML configuration + environment variables (§5).
- Color configuration (§5.3).
- Bash and fish completion (§6.6).
- Migrated tests (§7), man page, Makefile, goreleaser (§8).

### Out of scope

- **All addon functionality** (`.todo.actions.d`, `TODO_ACTIONS_DIR`): addon execution, `listaddons`, `help`/`shorthelp` with addon section, `command` (meaningless without addons).
- `deduplicate` and `listfile` (outside the MVP).
- `TODOTXT_SORT_COMMAND`, `TODOTXT_FINAL_FILTER`, effective `TODOTXT_DISABLE_FILTER`: they are shell `eval`'d commands, not replicable in Go. `-x` is accepted as a no-op for CLI compatibility (with default config it is already a no-op in todo.sh).
- `TODOTXT_SIGIL_BEFORE_PATTERN` / `SIGIL_VALID_PATTERN` / `SIGIL_AFTER_PATTERN`: POSIX BRE regexes not translatable 1:1 to RE2; the defaults (empty / `.*`) are hardcoded.
- Dynamic bash completion of the original todo_completion (contexts/projects/tasks are partially covered with cobra's ValidArgsFunction, §6.6).

## 3. Design decisions (agreed)

| Topic | Decision |
|---|---|
| Binary name | `gtdo` |
| Config format | Own TOML (BurntSushi/toml, like gitia) — bash is not parsed |
| Config location | `-d PATH` / `$GTDO_CONFIG` > `~/.config/gtdo/config.toml` > `/etc/gtdo/config.toml` |
| Precedence | CLI flags > env vars > TOML > defaults |
| Env vars | `TODO_DIR`, `TODO_FILE`, `DONE_FILE`, `REPORT_FILE`, `TODOTXT_*` keep working (scripting compatibility) |
| Colors | Always emitted except in plain mode (`-p` or config), same as todo.sh (no TTY detection) |
| Interactive prompts | Identical: `Add:`, `Append:`, `Delete '...'? (y/n)`; `-f` skips them |
| Addons | Removed entirely |

## 4. Architecture and layout

```
gtdo-cli/
├── cmd/gtdo/main.go          — entrypoint, signals, exit codes (gitia pattern)
├── internal/cli/             — cobra tree: root + actions, version, completions
│   └── testdata/script/*.txtar — black-box session tests
├── internal/todo/            — domain: Task, parsing, filters, sort, mutations, _format pipeline
├── internal/config/          — TOML + env vars + precedence
├── internal/ui/              — ANSI colors, output formatting
├── internal/exitcode/        — exit codes
├── tools/genman/             — man page generation
├── man/gtdo.1                — generated and committed man page
├── Makefile                  — build/test/lint/man/install/release (gitia pattern)
├── .goreleaser.yaml
├── go.mod, AGENTS.md, CHANGELOG.md, LICENSE, README.md
└── ACTIONS.md                — checklist (already exists)
```

Dependencies: `github.com/spf13/cobra`, `github.com/spf13/pflag`, `github.com/BurntSushi/toml`, `github.com/rogpeppe/go-internal` (tests), `golang.org/x/sys` (TTY if needed).

## 5. Configuration

### 5.1 Resolution

TOML file search order: `-d PATH` if passed, else `$GTDO_CONFIG`, else `~/.config/gtdo/config.toml`, else `/etc/gtdo/config.toml`. If none exists → defaults. Unlike todo.sh there is no fatal error if the file is missing (there is no mandatory config file).

### 5.2 TOML schema

```toml
[paths]
dir = "~/todo"              # TODO_DIR
todo_file = "~/todo/todo.txt"
done_file = "~/todo/done.txt"
report_file = "~/todo/report.txt"

[behavior]
force = false               # TODOTXT_FORCE
preserve_line_numbers = true  # TODOTXT_PRESERVE_LINE_NUMBERS
auto_archive = true         # TODOTXT_AUTO_ARCHIVE
date_on_add = false         # TODOTXT_DATE_ON_ADD
priority_on_add = ""        # TODOTXT_PRIORITY_ON_ADD (letter A-Z)
verbose = 1                 # TODOTXT_VERBOSE
default_action = ""         # TODOTXT_DEFAULT_ACTION
sourcevar = ""              # TODOTXT_SOURCEVAR (source file for listcon/listproj)
sentence_delimiters = ",.:;"  # SENTENCE_DELIMITERS

[colors]
pri_a = "\\033[1;33m"       # YELLOW
pri_b = "\\033[0;32m"       # GREEN
pri_c = "\\033[1;34m"       # LIGHT_BLUE
pri_x = "\\033[1;37m"       # WHITE
color_done = "\\033[0;37m"  # LIGHT_GREY
color_project = ""
color_context = ""
color_date = ""
color_number = ""
color_meta = ""

[colors.map]                # map of 16 ANSI colors (NONE, BLACK...WHITE, DEFAULT)
yellow = "\\033[1;33m"
...
```

Notes:
- The `[colors]` keys may reference names from the `[colors.map]` (e.g. `pri_a = "yellow"`) or direct ANSI codes.
- `$HOME` and `~` are expanded in paths.
- Env vars: `TODO_DIR`, `TODO_FILE`, `DONE_FILE`, `REPORT_FILE`, `TODOTXT_FORCE`, `TODOTXT_PRESERVE_LINE_NUMBERS`, `TODOTXT_AUTO_ARCHIVE`, `TODOTXT_DATE_ON_ADD`, `TODOTXT_PRIORITY_ON_ADD`, `TODOTXT_VERBOSE`, `TODOTXT_DEFAULT_ACTION`, `TODOTXT_SOURCEVAR`, `TODOTXT_PLAIN`, `SENTENCE_DELIMITERS`.
- Colors are configured **only via TOML** (todo.sh uses `export PRI_A=...` in bash; in gtdo color env vars are not supported in the MVP).

### 5.3 Precedence

1. CLI flags (todo.sh's `OVR_*`): `-a/-A`, `-c/-p`, `-f`, `-n/-N`, `-t/-T`, `-v`, `-x` (no-op).
2. `TODO_*` / `TODOTXT_*` env vars.
3. TOML.
4. todo.sh defaults: verbose=1, plain=0, force=0, preserve_line_numbers=1, auto_archive=1, date_on_add=0.

`-v` semantics replicated exactly: if the `TODOTXT_VERBOSE` env var is defined it wins; otherwise `max(1, count of -v)`. `-h` ≡ `shorthelp` action.

## 6. Behavior

### 6.1 Flags

- `-@` hides contexts (odd count) / shows (even count); `-+` same for projects; `-P` same for priority tags (the number of occurrences toggles).
- `-c` plain=0; `-p` plain=1.
- `-d PATH` alternate config; `-f` force; `-h` → shorthelp; `-n` preserve=0; `-N` preserve=1; `-t` date_on_add=1; `-T` date_on_add=0; `-v` verbose++ ; `-V` version (exit 0); `-x` no-op.
- Flags are accepted before the action (todo.sh-style getopts: `gtdo -p list`, not `gtdo list -p`). Cobra is configured to allow flags only before the subcommand.

### 6.2 Listing pipeline (`_format`)

1. **Numbering**: real line number from the file, right-aligned with padding to the width of the total line count (`sed =` + reformat; e.g. 10+ tasks → ` 1`, `10`).
2. **Filters** (`filtercommand`): each term AND'ed with `grep -i` (basic regex); `-TERM` → exclusion; `\|` inside a term → OR.
3. **Sort** (`LC_COLLATE=C sort -f -k2`): by task text case-insensitive; ties → original file order (numbers are zero-padded before sorting, which preserves original order).
4. **Colors** (todo.sh's awk): line `^[0-9]+ x ` → `color_done`; `^[0-9]+ \([A-Z]\) ` → `pri_<letter>` (fallback `pri_x`); words: number → `color_number`, `+foo` (ends in alphanumeric) → `color_project`, `@foo` → `color_context`, valid `(19|20)xx-xx-xx` date → `color_date`, `key:value` → `color_meta`. `-P` removes the `(X)` tag from output. `-@`/`-+` remove the sigils. Line color resets after each colored word (DEFAULT + base line color).
5. **Summary** (verbose > 0): `--` + `PREFIX: N of M tasks shown`, where PREFIX = basename of the file without extension, uppercased (`TODO` for todo.txt, `DONE` for done.txt).

### 6.3 Actions (todo.sh semantics)

- **add/addm/addto**: strip CR/LF; `add`/`addm` ask for interactive input if missing (`Add: `) unless `-f`; `addto DEST` requires the file to exist in the TODO_DIR. `date_on_add` prepends `YYYY-MM-DD `; `priority_on_add` prepends `(X) ` (after the date). Output: `N task` + `TODO: N added.` (verbose>0).
- **append**: prompts `Append: ` if text is missing (unless `-f`); space before the text unless it starts with a sentence delimiter (`,.:;` configurable via `SENTENCE_DELIMITERS`); escapes `\`, `|`, `&` for the sed substitution (net effect: the text is inserted literally).
- **prepend**: same without added space; preserves existing priority and date at the start (regex `priAndDateExpr`).
- **pri**: validates A-Z; replaces existing priority keeping the date; errors: `TODO: Invalid priority given. Must be capital A-Z.` / `TODO: No task $item.` / `TODO: $item already prioritized with (X).` (with `-f` it re-prioritizes).
- **do**: prepends `x YYYY-MM-DD ` (preserving priority); multiple NRs; auto-archive if `auto_archive` (moves `x ` lines to done.txt; verbose: `TODO: $TODO_FILE archived.`).
- **del**: confirmation `Delete '...'? (y/n) ` unless `-f` (answer `n` → `TODO: No tasks were deleted.` exit 1); with TERM deletes only the term (message `TODO: 'TERM' not found; no removal done.` exit 1); `preserve_line_numbers` leaves a blank line or compacts.
- **depri**: multiple NRs (also comma-separated); `TODO: $item no priority set.` if none.
- **move**: confirmation unless `-f`; validates destination; `TODO: No task $item in $SRC.` if it doesn't exist.
- **replace**: `TODO: No task $item.` if it doesn't exist.
- **archive**: moves `x ` to done.txt, removes blank lines, messages `TODO: $TODO_FILE archived.` / `TODO: $TODO_FILE does not contain any done tasks.`.
- **list/listall/listpri/listcon/listproj**: pipeline §6.2; `listall` concatenates todo.txt + done.txt; `listpri` accepts `A` or `A-C` (range); `listcon`/`listproj` list unique sigils (`sort -u`).
- **report**: writes `N open tasks` / `M done tasks` to report.txt (date included).
- **help/shorthelp**: gtdo's own texts (§6.4).

The exact error messages (`die` → stderr, exit 1) are extracted from todo.sh during implementation; the tests pin them byte for byte.

### 6.4 Help and version texts

- `usage` (unknown action or no action): `Usage: gtdo [-fhpantvV] [-d todo_config] action [task_number] [task_description]` + `Try 'gtdo -h' for more information.` → stdout, exit 1.
- `shorthelp` / `-h`: one-line list of actions (without addon section), with `gtdo` as the name.
- `help [ACTION...]`: gtdo full help + per-action; no addon section.
- `-V`: gtdo's own version text (name, version, repo), exit 0.
- cobra's `--help` is disabled/overridden so it doesn't clash with `-h` (which is shorthelp).

### 6.5 Other todo.sh behaviors

- Creates `TODO_DIR` (mkdir -p) and the todo/done/report files if they don't exist.
- `SENTENCE_DELIMITERS` defaults to `,.:;`.
- Dates: local `date +%Y-%m-%d`; tests pin `TZ=UTC`.
- `list` with `TODOTXT_SOURCEVAR` reading from another file (only listcon/listproj in todo.sh).
- `add`/`append`/etc. output with real line number.

### 6.6 Completion

- cobra `completion` (bash, fish) enabled in the binary.
- ValidArgsFunction to complete `@contexts` and `+projects` (from TODO_FILE) and task numbers where applicable (partial parity with todo_completion; the t6xxx tests are ported only for what cobra covers).

## 7. Testing

### 7.1 Session tests (txtar, go-internal)

They replicate the `test_todo_session` of the shell tests in scope: t1000 (add/list), t1010 (add-date), t1020/t1030 (addto), t1040 (add-priority), t1050 (todofile-override), t1100 (replace), t1200 (pri), t1250 (listpri), t1300 (ls), t1310 (listcon), t1320 (listproj), t1330 (ls-highlighting), t1340 (listescapes), t1350 (listall), t1360/t1380 (highlighting), t1400 (prepend), t1500 (do), t1600 (append), t1700 (depri), t1800 (del), t1850 (move), t1900 (archive), t1950 (report), t2000 (multiline), t2100/t2110/t2120 (help), t2200 (no-done-report-files), t0000 (config), t0001 (null), t0002 (actions/flags).

Each case: initial file state (txtar) + sequence of `gtdo ...` commands with expected stdout/exit. `TZ=UTC`, isolated `HOME`, no network.

### 7.2 Unit tests

Per package: `internal/todo` (priority/date parsing, filters, sort, mutations, pipeline), `internal/config` (path resolution, precedence, TOML), `internal/ui` (colors, padding, hide toggles).

### 7.3 Parity verification

During development: run the real todo.sh (in /tmp) and gtdo against the same fixtures, comparing outputs. The txtar tests are the permanent guarantee.

## 8. Extras (gitia pattern)

- **Makefile**: `build` (ldflags with version/commit/date), `test`, `lint` (golangci-lint), `man`, `install`, `release`/`release-dry` (goreleaser), `clean`.
- **Man page**: `tools/genman` + committed `man/gtdo.1`.
- **AGENTS.md** with repo conventions (like gitia, adapted to gtdo).
- **CHANGELOG.md** in Keep a Changelog format.
- **LICENSE** (MIT, same as todo.txt-cli).
- **.goreleaser.yaml** with per-platform builds and completions.

## 9. Acceptance criteria

1. `go test ./...` green.
2. All ported txtar tests pass with the same output as the original shell tests.
3. For a set of fixtures, `gtdo` and `todo.sh` produce identical stdout/stderr/exit codes/file states (verifiable with a comparison script during development).
4. `make build`, `make man`, `make lint` work.
5. No reference to addons remains in the code or the help.

## 10. Risks and notes

- The date regex and sigil patterns use BRE in todo.sh; in Go, RE2 equivalents carefully verified against the tests are used.
- GNU `sort -f` under `LC_COLLATE=C` compares byte by byte after lowercasing; the Go comparator must replicate ties by original line.
- The `Delete '...'? (y/n) ` prompt uses `read -N 1` (one char) in modern bash; in Go one char is read with optional Enter confirmation — verify against test t1800.
- `report` uses todo.sh's exact format (verify during implementation).
