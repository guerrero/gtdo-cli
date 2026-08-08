# AGENTS.md

Conventions for this repository. gtdo is a Go port of todo.txt-cli (todo.sh)
with a byte-parity contract; the design lives in
`docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.en.md` and the
migration checklist in ACTIONS.md.

## Build and test

- `make build` — build `./gtdo` with version metadata (ldflags).
- `make test` — run the full suite (`go test ./...`).
- `make lint` — golangci-lint (gofumpt formatting).
- `make man` — regenerate `man/gtdo.1`.
- `make release-dry` — goreleaser snapshot; `make release` — tagged release.

## Testing

- Session tests live in `internal/cli/testdata/script/*.txtar` and run through
  the testscript harness (go-internal). They pin stdout, stderr, exit codes,
  and file states byte for byte — treat them as the parity contract. The
  harness sets `TZ=UTC`, an isolated `HOME`, and `$ESC` for color tests.
- Unit tests per package: `internal/todo` (parse, filters, sort, mutations,
  pipeline), `internal/config` (resolution, precedence, TOML), `internal/ui`
  (colors, padding, hide toggles).
- The golden source for messages, prompts, and formats is
  `/tmp/todo.txt-cli/todo.sh`; when in doubt, run it and compare.

## Commit conventions

Conventional Commits v1.0.0. The subject is imperative, lower case, and has no
trailing period. The header stays within 72 characters.

Scope by package, without the `internal/` prefix: `cli`, `config`, `todo`,
`ui`, `exitcode`, `tools`. Use no scope for changes that span the whole
repository.

Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
`build`, `ci`, `chore`, `revert`.

Write a body when the change is not self-evident from the subject. Explain why,
not what — the diff already says what. Omit the body for mechanical changes.

## Code style

Standard Go with gofumpt. Package comments on every package. Comments explain
why a decision was made, not what the line does.

Output text (messages, prompts, help) and exit codes are part of the parity
contract: any change requires updating the txtar expectations and re-checking
against todo.sh. No addon references anywhere — `command`, `deduplicate`,
`listfile`, and `listaddons` are out of scope.

## Release

Releases are manual. Version numbers follow Semantic Versioning (pre-1.0:
`feat` → minor, `fix` → patch). The changelog is curated by hand in Keep a
Changelog format: every release moves the `[Unreleased]` section to a dated
version heading. Tag `vX.Y.Z` on `main`, then
`GITHUB_TOKEN=$(gh auth token) make release` publishes the assets.
