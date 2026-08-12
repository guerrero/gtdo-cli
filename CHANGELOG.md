# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Configurable `format` action for normalizing todo.txt and done.txt task fields.

- Added opt-in `enableUUID` / `GTDO_ENABLE_UUID` timestamp IDs for new tasks,
  formatted as `YYYYMMDDTHHMMSS.nnZ`. IDs advance in 10 ms steps when a
  collision is detected and remain stable through edits and moves.

- Guided `add -g` now asks for an optional priority (`--only priority`) and
  runs its phases in task, priority, context, project, metadata order.

### Changed

- Replaced TOML configuration with a strict camelCase JSON format at
  `~/.config/gtdo/config.json`.

### Removed

- Retired the legacy date-on-add behavior, including its `date_on_add` setting,
  `TODOTXT_DATE_ON_ADD` environment variable, and `-t` / `-T` options.

## [0.1.0] - 2026-08-09

### Added

- Go port of todo.txt-cli with 20 built-in actions and byte-parity session tests.
- TOML configuration, environment-variable compatibility, ANSI colors, and shell completions.
- Bash/fish completion archives, generated man page, parity verification script, and release builds for six platforms.
