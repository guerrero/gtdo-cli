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

---

# Execution plan (SDD)

## Global constraints (bind every task)

- **Byte parity**: for a given input, stdout, stderr, exit code, and resulting file state must be identical to todo.sh (todo.txt-cli v2.x), except §6.4 help/version texts and §2 exclusions. No addon support anywhere.
- Reference sources available at: `/tmp/todo.txt-cli/` (todo.sh, todo.cfg, tests/, todo_completion) and `/Users/alex/Proyectos/Personales/gitia/` (structure pattern).
- Module: `github.com/guerrero/gtdo`; binary `gtdo`. Repo layout per §4. Deps: cobra, pflag, BurntSushi/toml, rogpeppe/go-internal (tests), golang.org/x/sys.
- Global flags: `-@ -+ -a -A -c -d -f -h -n -N -p -P -t -T -v -V -x`; flags only before the action (getopts style), never after (§6.1).
- Exact messages, prompts and exit codes come from the real todo.sh (`/tmp/todo.txt-cli/todo.sh`); tests pin them byte for byte.
- TZ=UTC and isolated HOME in txtar tests; `$ESC` env expands to a real ESC byte for color tests.
- No references to addons in code or help. `command`, `deduplicate`, `listfile`, `listaddons` absent.
- All commits on the current branch (`resolute-moose`). Keep `go test ./...` green after each task.

## Task 1: Repo scaffolding completion

Complete the repository skeleton per §4 and §8 (gitia pattern). The scaffold commit `2038c1e` already provides: go.mod, Makefile, .golangci.yml, .editorconfig, .gitignore, cmd/gtdo/main.go, internal/cli (preparse.go, root.go, usage.go, version.go, script_test.go + txtar harness with usage.txtar/version.txtar), internal/config (Options skeleton only), internal/exitcode.

Still missing, to be created in this task:
- `internal/todo/` package skeleton (package doc comment only; no behavior yet) with files the next tasks will fill: task.go, parse.go, filter.go, sort.go, mutate.go, format.go (doc-comment stubs are fine).
- `internal/ui/` package skeleton with doc comment.
- `internal/config/`: full implementation is Task 2 — do not implement here beyond the existing Options.
- `tools/genman/` stub is NOT needed yet (Task 10).
- Root files: `LICENSE` (MIT, same as todo.txt-cli), `README.md` (project intro, usage, build instructions), `AGENTS.md` (repo conventions, adapted from gitia's), `CHANGELOG.md` (Keep a Changelog format, "Unreleased" section), `.goreleaser.yaml` (builds per platform: darwin/linux/windows amd64+arm64, archives with completion scripts — bash/fish; mirror gitia's if present).
- `ACTIONS.md` already exists at repo root; leave as-is.

Acceptance:
- `make build` produces `./gtdo`; `./gtdo -V` prints the version block and exits 0.
- `go test ./...` green (existing preparse/script tests pass).
- `make lint` runs (golangci-lint may not be installed; `make lint` must at least be invocable — if golangci-lint is missing from PATH, note it in the report instead of failing the task).
- README mentions the 20 actions, the config file, and the parity goal.

## Task 2: internal/config — TOML, env vars, precedence

Implement `internal/config` fully per §5:
- TOML schema §5.2: `[paths]` dir/todo_file/done_file/report_file; `[behavior]` force, preserve_line_numbers, auto_archive, date_on_add, priority_on_add, verbose, default_action, sourcevar, sentence_delimiters; `[colors]` pri_a..pri_x, color_done, color_project, color_context, color_date, color_number, color_meta; `[colors.map]` 16 ANSI color names (NONE, BLACK, RED, GREEN, YELLOW, BLUE, MAGENTA, CYAN, WHITE, DEFAULT + bright variants as in todo.cfg). Use BurntSushi/toml.
- File resolution §5.1: `-d PATH` (Options.ConfigPath from pre-parser) > `$GTDO_CONFIG` > `~/.config/gtdo/config.toml` > `/etc/gtdo/config.toml`; missing file → defaults (no fatal error).
- Env vars §5.2: TODO_DIR, TODO_FILE, DONE_FILE, REPORT_FILE, TODOTXT_FORCE, TODOTXT_PRESERVE_LINE_NUMBERS, TODOTXT_AUTO_ARCHIVE, TODOTXT_DATE_ON_ADD, TODOTXT_PRIORITY_ON_ADD, TODOTXT_VERBOSE, TODOTXT_DEFAULT_ACTION, TODOTXT_SOURCEVAR, TODOTXT_PLAIN, SENTENCE_DELIMITERS.
- Precedence §5.3: CLI flags (Options.*Set from preparse) > env vars > TOML > todo.sh defaults (verbose=1, plain=0, force=0, preserve_line_numbers=1, auto_archive=1, date_on_add=0, sentence_delimiters=",.:;").
- `-v` semantics §5.3: if TODOTXT_VERBOSE env is defined it wins; else verbose = max(1, number of -v occurrences).
- Path expansion: `~` and `$HOME` expanded in paths from TOML/env.
- Colors resolvable by `[colors.map]` name (e.g. `pri_a = "yellow"`) or direct ANSI code string.
- Colors configured only via TOML (no color env vars).

Unit tests (`internal/config/config_test.go`): precedence (flag wins over env over TOML over default, each layer), file search order, env var parsing (bool/int/string), `~`/`$HOME` expansion, TOML parse of full schema, color name resolution vs direct ANSI, missing file → defaults, `-v` counting semantics.

Acceptance: `go test ./internal/config/` green; `go vet ./internal/config/` clean.

## Task 3: internal/todo — task model, parse, filters, sort

Implement the domain core per §6.2 (pipeline steps 1–3 primitives) and §6.3 semantics where they concern parsing:
- `Task` type: raw line text, real file line number (1-based), parsed priority (A-Z or none), parsed date (YYYY-MM-DD at line start), done flag (`x ` prefix), contexts `@...`, projects `+...` (sigil word ends in alphanumeric per §6.2.4).
- Parsing per todo.sh conventions: priority regex `^\([A-Z]\) `, date regex `(19|20)\d\d-\d\d-\d\d`, `x ` done marker (see todo.sh `_get_priority`/`_get_date`/`_get_done`); use RE2 equivalents verified against todo.sh tests.
- Filters (§6.2.2): terms AND'ed; `-TERM` excludes; `\|` inside a term is OR; case-insensitive; basic regex (grep -i semantics — translate to RE2 carefully; escape where grep basic regex differs).
- Sort (§6.2.3 + §10): case-insensitive by task text; ties keep original file order (zero-padded line numbers before comparing).
- Line numbering padding (§6.2.1): right-aligned to width of total line count.

Unit tests: parse (priority/date/done/contexts/projects, edge cases), filters (AND, OR via \|, exclusion, case), sort (case-insensitive, ties stable, padding). Cross-check behaviors against `/tmp/todo.txt-cli/tests/` where relevant.

Acceptance: `go test ./internal/todo/` green.

## Task 4: internal/todo — mutations and _format pipeline

Implement mutations and the listing pipeline per §6.2–§6.3:
- Mutations (all with todo.sh-exact semantics; consult `/tmp/todo.txt-cli/todo.sh`): add (CR/LF strip, optional date_on_add `YYYY-MM-DD ` and priority_on_add `(X) ` prefixes, returns new line number), addm (multi-line add), addto (DEST must exist inside TODO_DIR), append (space before text unless it starts with a sentence delimiter from SENTENCE_DELIMITERS; escape `\`, `|`, `&` for literal insert), prepend (no added space; preserves existing priority+date prefix via todo.sh's priAndDateExpr), pri (validate A-Z, replace existing priority keeping date), depri (multiple NRs, comma-separated too), do (prefix `x YYYY-MM-DD ` keeping priority; multiple NRs), del (NR or TERM within task; preserve_line_numbers leaves blank line), replace, move (between files in TODO_DIR).
- File store: create TODO_DIR (mkdir -p) and todo/done/report files if missing (§6.5); real line numbers from the file.
- `_format` pipeline (§6.2): numbering → filters → sort → colors (§6.2.4: done line → color_done; `^[0-9]+ \([A-Z]\) ` → pri_<letter> fallback pri_x; words: number → color_number, +project → color_project, @context → color_context, valid date → color_date, key:value → color_meta; reset to DEFAULT + base line color after each colored word) → summary (`--` + `PREFIX: N of M tasks shown`, PREFIX = uppercase basename without extension). Plain mode disables colors. `-@`/`-+`/`-P` hide sigils/priority label.
- Colors themselves come from config; this task defines the color hooks (plain struct of ANSI strings, populated by config in Task 2 wiring — keep the interface minimal).

Unit tests: each mutation (happy path + error cases), archive logic, append delimiter/escape behavior, prepend priAndDate preservation, del preserve_line_numbers, pipeline (numbering/filter/sort/colors/summary, hide toggles).

Acceptance: `go test ./internal/todo/` green.

## Task 5: internal/ui — colors and formatting

Implement `internal/ui` per §5.2 (colors) and §6.2.4:
- The 16-name ANSI color map (NONE/BLACK/RED/GREEN/YELLOW/BLUE/MAGENTA/CYAN/WHITE/DEFAULT + bright variants, values matching todo.cfg).
- Render helpers: apply line color, per-word coloring with reset, plain mode short-circuit, number padding, hide sigils/priority helpers used by the pipeline.
- Color struct: `{PriA, PriB, PriC, PriX, Done, Project, Context, Date, Number, Meta string}` — plain (no escape sequences) when plain mode.

Unit tests: color resolution (name vs ANSI code), reset behavior after colored words, padding, hide toggles, plain mode.

Acceptance: `go test ./internal/ui/` green.

## Task 6: internal/cli — root wiring, flags, help/version/usage texts

Wire the CLI layer per §6.1, §6.4, §6.5:
- Verify/extend the pre-parser against §6.1 semantics (scaffold preparse.go already covers most; check `-h` → shorthelp with remaining args discarded, `-V` → version, `-d` with attached or separate arg, `--` handling).
- Cobra root: `DisableFlagParsing` (pre-parser owns flags), `SilenceUsage`, `SilenceErrors`, exit codes via internal/exitcode (0 ok, 1 failure).
- Texts §6.4 (exact, gtdo's own):
  - usage: `Usage: gtdo [-fhpantvV] [-d todo_config] action [task_number] [task_description]` + `Try 'gtdo -h' for more information.` → stdout, exit 1 (no action or unknown action).
  - shorthelp `-h`: one-line list of actions (no addon section).
  - help [ACTION...]: full help + per-action; no addon section.
  - `-V`: gtdo version text (name, version, repo), exit 0.
- Unknown action → usage text, exit 1 (stdout per §6.4).
- Default action from config (TODOTXT_DEFAULT_ACTION) applied when no action given.
- No cobra `--help` clash: `-h` is shorthelp; disable/override cobra's help flag on root (the pre-parser consumes `-h` before cobra sees it; make sure cobra never sees `-h`).
- txtar tests: usage.txtar (no action, unknown action, unknown flag), version.txtar (already exists), shorthelp.txtar, help.txtar.

Acceptance: `go test ./internal/cli/` green; manual spot-check `./gtdo` and `./gtdo -h` match §6.4 exactly.

## Task 7: Session actions — mutating actions

Implement the mutating actions as cobra subcommands wired to internal/todo, with todo.sh-exact behavior and messages (§6.3):
- `add` (alias `a`), `addm`, `addto`, `append` (alias `app`), `prepend` (alias `prep`), `replace`, `pri` (alias `p`), `depri` (alias `dp`), `do` (alias `done`), `del` (alias `rm`), `move` (alias `mv`), `archive`, `report`.
- Interactive prompts identical to todo.sh: `Add: `, `Append: `, `Delete '...'? (y/n) `; `-f` skips them. The delete prompt reads one char (verify t1800 expectations; optional Enter confirmation must not break parity with the shell test harness — check how the txtar harness supplies stdin).
- Exact messages (from todo.sh, verify each against `/tmp/todo.txt-cli/todo.sh`): `TODO: N added.`, `TODO: Invalid priority given. Must be capital A-Z.`, `TODO: No task $item.`, `TODO: $item already prioritized with (X).`, `TODO: $item no priority set.`, `TODO: No tasks were deleted.` (exit 1), `TODO: 'TERM' not found; no removal done.` (exit 1), `TODO: No task $item in $SRC.`, `TODO: $TODO_FILE archived.`, `TODO: $TODO_FILE does not contain any done tasks.`, addto missing-dest error.
- report writes `N open tasks` / `M done tasks` lines to report.txt with date (verify exact format in todo.sh `report()`).
- Auto-archive in `do` when auto_archive on; `archive` moves `x ` lines and removes blank lines.

txtar tests ported from `/tmp/todo.txt-cli/tests/`: t1000, t1010, t1020, t1030, t1040, t1050, t1100, t1200, t1400, t1500, t1600, t1700, t1800, t1850, t1900, t1950, t2000. Each: txtar fixture of initial files + commands with expected stdout/exit. Use `cmp` semantics via the testscript harness (want stdout files; use `$ESC` for colors). TZ=UTC, HOME isolated. The existing script_test.go harness already sets this up.

Acceptance: `go test ./internal/cli/` green with the ported tests; outputs byte-identical to running the real todo.sh tests (spot-check a few against `/tmp/todo.txt-cli/tests/` by running `bash /tmp/todo.txt-cli/todo.sh` equivalents — note todo.sh requires a config; use `-d` with the repo's todo.cfg or set env vars).

## Task 8: Listing actions + highlighting

Implement the listing actions per §6.2–§6.3:
- `list` (alias `ls`), `listall` (alias `lsa`), `listpri` (alias `lsp`, accepts `A` or `A-C` range), `listcon` (alias `lsc`), `listproj` (alias `lsprj`).
- `listall` concatenates todo.txt + done.txt; listcon/listproj list unique sigils (`sort -u`); listpri filters by priority range with optional filter terms.
- Wire the `_format` pipeline (§6.2) with numbering, filters, sort, colors, summary (`--` + `PREFIX: N of M tasks shown`); plain mode and `-@`/`-+`/`-P` toggles; TODOTXT_SOURCEVAR reads tasks from another file for listcon/listproj (§6.5).
- Highlighting per §6.2.4 with the ui package.

txtar tests: t1250, t1300, t1310, t1320, t1330, t1340, t1350, t1360, t1380, t2200, t0002 (actions/flags). Check each test in `/tmp/todo.txt-cli/tests/` for exact expected output including colors (the shell tests compare with `diff` against expected files; port the expected outputs using `$ESC`).

Acceptance: `go test ./internal/cli/` green.

## Task 9: Config/help/misc session tests + parity script

- Port remaining session tests: t0000 (config), t0001 (null), t2100/t2110/t2120 (help).
- Add a parity verification script (e.g. `scripts/parity.sh` or documented in the report) that runs the real todo.sh (from /tmp/todo.txt-cli, with a generated config) and gtdo against the same fixtures and diffs stdout/stderr/exit codes/file states (§7.3). The script must be runnable and used to verify at least the add/list/pri/do/del flows.
- Fix any byte differences found between gtdo and todo.sh during parity checks.

Acceptance: `go test ./...` green; parity script shows zero diffs for the checked flows (document the command and output in the report).

## Task 10: Completions, man page, release extras

- Completion per §6.6: cobra `completion` (bash, fish) enabled in the binary; ValidArgsFunction completing `@contexts` and `+projects` from TODO_FILE and task numbers where applicable.
- Man page per §8: `tools/genman` (cobra doc-gen based, gitia pattern) generating `man/gtdo.1`, committed; `make man` works.
- Verify `make build`, `make man`, `make lint` (if golangci-lint available), `make release-dry` (if goreleaser available) — report which tools were available.
- Final acceptance §9: `go test ./...` green; no addon references anywhere in code or help; `make build` works.
