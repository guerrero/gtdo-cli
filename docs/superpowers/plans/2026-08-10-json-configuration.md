# JSON Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace gtdo's TOML configuration with a strict, camelCase, JSON-only configuration format without changing runtime defaults or precedence.

**Architecture:** Split the on-disk schema/decoder from file discovery: `json.go` owns typed JSON decoding and validation, while `loader.go` owns search and file I/O. Keep the resolved `Config` API and color role API unchanged, then convert every executable fixture and user-facing document in the same migration.

**Tech Stack:** Go 1.26.5, standard-library `encoding/json`, existing go-internal testscript harness, generated roff man page.

## Global Constraints

- JSON is a hard switch: do not discover or parse legacy TOML configuration and do not add aliases or a converter.
- Search order is `-d PATH`, `$GTDO_CONFIG`, `~/.config/gtdo/config.json`, `/etc/gtdo/config.json`; explicit paths may have any filename but must contain JSON.
- The only top-level JSON properties are `dir`, `files`, `behaviour`, and `colors`.
- `files` contains `todo`, `done`, and `report`; `dir` remains outside `files`.
- Compound JSON property names are camelCase; use British `behaviour` and American `colors` exactly.
- Unknown properties, malformed or trailing JSON, type mismatches, `null`, and incorrect nesting are fatal parse errors.
- Preserve existing CLI > environment > JSON > default precedence, `-v` semantics, defaults, path expansion, color resolution, output, exit codes, and resulting file states.
- Use only Go's standard `encoding/json`; remove `github.com/BurntSushi/toml` and add no replacement dependency.

## File Structure

- Create `internal/config/json.go` — typed on-disk JSON schema plus strict decoding and `null` rejection.
- Create `internal/config/loader.go` — config discovery, file reading, decode error wrapping, and runtime resolution.
- Delete `internal/config/toml.go` — remove the old combined TOML schema/loader.
- Modify `internal/config/config.go` — resolve the new schema while keeping exported `Config` fields and methods stable.
- Modify `internal/config/defaults.go` — prefill JSON schema defaults.
- Modify `internal/config/colors.go` — consume `colorsJSON` while retaining internal snake_case role identifiers used by the UI.
- Modify `internal/config/config_test.go` — convert all resolution/discovery tests to JSON.
- Create `internal/config/json_test.go` — focused strict-decoder tests.
- Modify config-using CLI/UI tests and `scripts/parity.sh` — replace TOML fixtures with JSON without changing expected behavior.
- Modify live docs (`README.md`, `AGENTS.md`, `CHANGELOG.md`, `man/gtdo.1.tmpl`, generated `man/gtdo.1`) — publish the breaking format contract.
- Modify normative specs/plans that still prescribe TOML — keep future work consistent with the new schema.

---

### Task 1: Switch the parser and every executable fixture atomically

**Files:**
- Create: `internal/config/json.go`
- Create: `internal/config/json_test.go`
- Create: `internal/config/loader.go`
- Delete: `internal/config/toml.go`
- Modify: `internal/config/config.go:1-134`
- Modify: `internal/config/defaults.go:1-31`
- Modify: `internal/config/colors.go:5-31`
- Modify: `internal/config/config_test.go:1-629`
- Modify: `internal/ui/color_test.go:1-233`
- Modify: `internal/ui/color.go:1-12`
- Modify: `internal/cli/list.go:1-10`
- Modify: `internal/cli/multiline_test.go:20-25`
- Modify: `internal/cli/preparse_test.go:70-89`
- Modify: `internal/cli/script_test.go:25-34`
- Modify: `internal/cli/testdata/script/defaultaction-config.txtar`
- Modify: `internal/cli/testdata/script/t0000-config.txtar`
- Modify: `internal/cli/testdata/script/t1010-add-date.txtar`
- Modify: `internal/cli/testdata/script/t1030-addto-date.txtar`
- Modify: `internal/cli/testdata/script/t1040-add-priority.txtar`
- Modify: `internal/cli/testdata/script/t1330-ls-highlighting.txtar`
- Modify: `internal/cli/testdata/script/t1360-ls-project-context-highlighting.txtar`
- Modify: `internal/cli/testdata/script/t1380-ls-date-number-metadata-highlighting.txtar`
- Modify: `internal/cli/testdata/script/t1600-append.txtar`
- Modify: `internal/cli/testdata/script/shorthelp.txtar`
- Modify: `scripts/parity.sh:1-130`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: existing `Options`, `Config`, environment helpers, path helpers, `resolveColors`, and todo.sh-compatible defaults.
- Produces: `decodeFileConfig(data []byte, dst *fileConfig) error`, `findConfigFile(opts Options, home, systemPath string) string`, and `load(opts Options, home, systemPath string) (Config, error)`.
- Preserves: `Load(opts Options) (Config, error)`, `Config.Color(role string) string`, and `Config.PriorityColor(letter byte) string` exactly.

- [ ] **Step 1: Add focused tests that state the new strict JSON contract**

  Create `internal/config/json_test.go` with table-driven failures and a successful complete decode. Use fresh defaults for every case so one decode cannot contaminate another:

  ```go
  package config

  import (
      "strings"
      "testing"
  )

  func TestDecodeFileConfig(t *testing.T) {
      cfg := defaultFileConfig()
      err := decodeFileConfig([]byte(`{
        "dir":"~/todo",
        "files":{"todo":"todo.txt","done":"done.txt","report":"report.txt"},
        "behaviour":{"force":true,"preserveLineNumbers":false,"autoArchive":false,"dateOnAdd":true,"priorityOnAdd":"B","verbose":2,"defaultAction":"list","sourceVar":"done.txt","sentenceDelimiters":".;"},
        "colors":{"priA":"yellow","priZ":"cyan","colorDone":"light_grey","colorProject":"light_cyan","colorContext":"blue","colorDate":"red","colorNumber":"green","colorMeta":"brown","map":{"yellow":"\\033[1;43m"}}
      }`), &cfg)
      if err != nil {
          t.Fatal(err)
      }
      if cfg.Dir != "~/todo" || cfg.Files.Todo != "todo.txt" || cfg.Files.Done != "done.txt" || cfg.Files.Report != "report.txt" {
          t.Fatalf("paths = %#v/%q, want complete JSON paths", cfg.Files, cfg.Dir)
      }
      if !cfg.Behaviour.Force || cfg.Behaviour.PreserveLineNumbers || cfg.Behaviour.Verbose != 2 {
          t.Fatalf("behaviour = %#v, want decoded values", cfg.Behaviour)
      }
      if cfg.Colors.PriZ != "cyan" || cfg.Colors.Map["yellow"] != `\033[1;43m` {
          t.Fatalf("colors = %#v, want decoded values", cfg.Colors)
      }
  }

  func TestDecodeFileConfigRejectsInvalidDocuments(t *testing.T) {
      cases := map[string]string{
          "malformed":          `{"dir":`,
          "trailing value":     `{} {}`,
          "unknown top level": `{"paths":{}}`,
          "unknown nested":    `{"behaviour":{"preserve_line_numbers":true}}`,
          "wrong type":        `{"behaviour":{"verbose":"loud"}}`,
          "null scalar":       `{"dir":null}`,
          "null object":       `{"files":null}`,
          "null map value":    `{"colors":{"map":{"red":null}}}`,
          "incorrect nesting": `{"files":{"todo":{"file":"todo.txt"}}}`,
      }
      for name, body := range cases {
          t.Run(name, func(t *testing.T) {
              cfg := defaultFileConfig()
              err := decodeFileConfig([]byte(body), &cfg)
              if err == nil {
                  t.Fatal("decodeFileConfig accepted invalid JSON")
              }
              if name == "unknown nested" && !strings.Contains(err.Error(), "unknown field") {
                  t.Fatalf("error = %q, want unknown-field diagnostic", err)
              }
          })
      }
  }
  ```

- [ ] **Step 2: Run the new tests and confirm they fail for the missing JSON API**

  Run: `go test ./internal/config -run 'TestDecodeFileConfig' -count=1`

  Expected: FAIL to compile because `decodeFileConfig`, `Files`, and `Behaviour` do not exist yet.

- [ ] **Step 3: Add the typed JSON schema and strict decoder**

  Create `internal/config/json.go`. Define every accepted key explicitly; no catch-all maps except the intentional color-name map:

  ```go
  package config

  import (
      "bytes"
      "encoding/json"
      "errors"
  )

  type filesJSON struct {
      Todo   string `json:"todo"`
      Done   string `json:"done"`
      Report string `json:"report"`
  }

  type behaviourJSON struct {
      Force               bool   `json:"force"`
      PreserveLineNumbers bool   `json:"preserveLineNumbers"`
      AutoArchive         bool   `json:"autoArchive"`
      DateOnAdd           bool   `json:"dateOnAdd"`
      PriorityOnAdd       string `json:"priorityOnAdd"`
      Verbose             int    `json:"verbose"`
      DefaultAction       string `json:"defaultAction"`
      SourceVar           string `json:"sourceVar"`
      SentenceDelimiters  string `json:"sentenceDelimiters"`
  }

  type colorsJSON struct {
      PriA string `json:"priA"`
      PriB string `json:"priB"`
      PriC string `json:"priC"`
      PriD string `json:"priD"`
      PriE string `json:"priE"`
      PriF string `json:"priF"`
      PriG string `json:"priG"`
      PriH string `json:"priH"`
      PriI string `json:"priI"`
      PriJ string `json:"priJ"`
      PriK string `json:"priK"`
      PriL string `json:"priL"`
      PriM string `json:"priM"`
      PriN string `json:"priN"`
      PriO string `json:"priO"`
      PriP string `json:"priP"`
      PriQ string `json:"priQ"`
      PriR string `json:"priR"`
      PriS string `json:"priS"`
      PriT string `json:"priT"`
      PriU string `json:"priU"`
      PriV string `json:"priV"`
      PriW string `json:"priW"`
      PriX string `json:"priX"`
      PriY string `json:"priY"`
      PriZ string `json:"priZ"`

      ColorDone    string `json:"colorDone"`
      ColorProject string `json:"colorProject"`
      ColorContext string `json:"colorContext"`
      ColorDate    string `json:"colorDate"`
      ColorNumber  string `json:"colorNumber"`
      ColorMeta    string `json:"colorMeta"`
      Map          map[string]string `json:"map"`
  }

  type fileConfig struct {
      Dir       string        `json:"dir"`
      Files     filesJSON     `json:"files"`
      Behaviour behaviourJSON `json:"behaviour"`
      Colors    colorsJSON    `json:"colors"`
  }

  func decodeFileConfig(data []byte, dst *fileConfig) error {
      var raw any
      if err := json.Unmarshal(data, &raw); err != nil {
          return err
      }
      if containsJSONNull(raw) {
          return errors.New("null is not allowed in configuration")
      }

      decoder := json.NewDecoder(bytes.NewReader(data))
      decoder.DisallowUnknownFields()
      return decoder.Decode(dst)
  }

  func containsJSONNull(value any) bool {
      switch value := value.(type) {
      case nil:
          return true
      case []any:
          for _, item := range value {
              if containsJSONNull(item) {
                  return true
              }
          }
      case map[string]any:
          for _, item := range value {
              if containsJSONNull(item) {
                  return true
              }
          }
      }
      return false
  }
  ```

  `json.Unmarshal` performs the first full-document pass, so malformed JSON and multiple/trailing values fail before the strict typed pass. `DisallowUnknownFields` then rejects unknown keys at every typed nesting level.

- [ ] **Step 4: Split discovery/loading from the schema and switch default filenames**

  Delete `internal/config/toml.go` after moving its discovery behavior into `internal/config/loader.go`:

  ```go
  package config

  import (
      "fmt"
      "os"
      "path/filepath"
  )

  const systemConfigPath = "/etc/gtdo/config.json"

  func findConfigFile(opts Options, home, systemPath string) string {
      candidates := []string{opts.ConfigPath}
      if value, ok := envString("GTDO_CONFIG"); ok {
          candidates = append(candidates, expandHome(value, home))
      }
      candidates = append(candidates,
          filepath.Join(home, ".config", "gtdo", "config.json"),
          systemPath,
      )
      for _, path := range candidates {
          if path != "" && exists(path) {
              return path
          }
      }
      return ""
  }

  func load(opts Options, home, systemPath string) (Config, error) {
      path := findConfigFile(opts, home, systemPath)
      file := defaultFileConfig()
      if path != "" {
          data, err := os.ReadFile(path)
          if err != nil {
              return Config{}, fmt.Errorf("config: read %s: %w", path, err)
          }
          if err := decodeFileConfig(data, &file); err != nil {
              return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
          }
      }
      cfg := resolve(opts, file, home)
      cfg.ConfigPath = path
      return cfg, nil
  }
  ```

- [ ] **Step 5: Rewire defaults, resolution, and color schema types**

  In `internal/config/defaults.go`, return `behaviourJSON` and `colorsJSON` while keeping every value unchanged:

  ```go
  func defaultFileConfig() fileConfig {
      return fileConfig{
          Behaviour: behaviourJSON{
              PreserveLineNumbers: true,
              AutoArchive:         true,
              Verbose:             1,
              SentenceDelimiters:  ",.:;",
          },
          Colors: defaultColors(),
      }
  }

  func defaultColors() colorsJSON {
      return colorsJSON{
          PriA: "yellow", PriB: "green", PriC: "light_blue",
          PriX: "white", ColorDone: "light_grey",
      }
  }
  ```

  In `internal/config/config.go`, replace `f.Paths.Dir` with `f.Dir`, `f.Paths.TodoFile`/`DoneFile`/`ReportFile` with `f.Files.Todo`/`Done`/`Report`, and every `f.Behavior` with `f.Behaviour`. Update comments to say JSON, `files`, and `behaviour`; do not rename the exported Go field `Config.DefaultAction` or any other runtime field.

  In `internal/config/colors.go`, change receivers and parameters from `colorsTOML` to `colorsJSON`. Keep the maps returned by `roles()` keyed as `pri_a`, `color_done`, and the other existing snake_case runtime role names because `internal/ui` consumes those identifiers, not the JSON property names.

- [ ] **Step 6: Convert the existing config resolution suite to JSON**

  In `internal/config/config_test.go`:

  - Change helper-created filenames and nonexistent system paths from `config.toml` to `config.json`.
  - Rename `TestTOMLFullSchema` to `TestJSONFullSchema` and `TestInvalidTOML` to `TestInvalidJSON`.
  - Replace format labels in assertion messages with `JSON`.
  - Convert each body to the exact new nesting and key names. For example:

  ```go
  body := `{
    "dir": "~/todo",
    "files": {
      "todo": "~/todo/todo.txt",
      "done": "~/todo/done.txt",
      "report": "~/todo/report.txt"
    },
    "behaviour": {
      "force": true,
      "preserveLineNumbers": false,
      "autoArchive": false,
      "dateOnAdd": true,
      "priorityOnAdd": "B",
      "verbose": 2,
      "defaultAction": "list",
      "sourceVar": "~/done.txt",
      "sentenceDelimiters": ".;"
    },
    "colors": {
      "priA": "yellow",
      "priB": "\\033[0;32m",
      "colorProject": "light_cyan",
      "colorMeta": "",
      "map": {"yellow": "\\033[1;43m"}
    }
  }`
  ```

  For table-driven one-setting bodies, use `fmt.Sprintf` with complete JSON objects rather than string-concatenating fragments:

  ```go
  body := fmt.Sprintf(`{"behaviour":{"%s":true}}`, tc.jsonKey)
  body := fmt.Sprintf(`{"behaviour":{"%s":"json"}}`, tc.jsonKey)
  ```

  Rename table fields such as `tomlKey`/`useTOML` to `jsonKey`/`useJSON`. Convert path tests to `{"dir":"~/todo","files":{...}}` and color tests to `{"colors":{"priA":"yellow"}}` forms.

  Add a discovery subtest proving `~/.config/gtdo/config.toml` is ignored when no JSON candidate exists:

  ```go
  t.Run("legacy home toml is ignored", func(t *testing.T) {
      h := home(t)
      writeConfig(t, filepath.Join(h, ".config", "gtdo", "config.toml"), `{"behaviour":{"verbose":9}}`)
      cfg := loadWith(t, Options{}, h)
      if cfg.ConfigPath != "" || cfg.Verbose != 1 {
          t.Fatalf("ConfigPath = %q, Verbose = %d; want empty, 1", cfg.ConfigPath, cfg.Verbose)
      }
  })
  ```

- [ ] **Step 7: Run the config package tests and vet the core switch**

  Run: `gofmt -w internal/config/json.go internal/config/json_test.go internal/config/loader.go internal/config/config.go internal/config/defaults.go internal/config/colors.go internal/config/config_test.go`

  Run: `go test ./internal/config -count=1`

  Expected: PASS, including strict validation, search order, precedence, path expansion, and color behavior.

  Run: `go vet ./internal/config/`

  Expected: no output and exit 0.

- [ ] **Step 8: Convert every CLI and UI fixture that loads configuration**

  Change harness defaults in `internal/cli/script_test.go` and `internal/cli/multiline_test.go` to `gtdo-config.json`. The filename-only parser tests in `internal/cli/preparse_test.go` should use `cfg.json`; their expected `Options.ConfigPath` remains the literal input.

  Convert each listed txtar embedded file and matching `-d` argument from `.toml` to `.json`. Use these exact mappings:

  ```json
  {"behaviour":{"defaultAction":"shorthelp"}}
  {"behaviour":{"dateOnAdd":true}}
  {"behaviour":{"priorityOnAdd":"A"}}
  {"behaviour":{"sentenceDelimiters":"&%"}}
  {"colors":{"priA":"yellow","colorProject":"red"}}
  ```

  For `t0000-config.txtar`, represent the full fixture with top-level `dir`, `files`, and `behaviour`; copy it to `.config/gtdo/config.json`, `alt.json`, and `env.json`. Keep the missing explicit path fall-through case, but name it `nope.json`. Add or preserve a legacy `.config/gtdo/config.toml` embedded file whose JSON-looking contents would change behavior if discovered, and assert it is ignored.

  Rewrite `internal/ui/color_test.go`'s `loadConfig` helper to write `config.json`, and convert its bodies to `colors` JSON objects with camelCase keys. Update comments in `internal/ui/color.go` and `internal/cli/list.go` to distinguish JSON keys (`priA`) from resolved runtime roles (`pri_a`).

- [ ] **Step 9: Convert the parity harness configuration**

  In `scripts/parity.sh`, write `$WORK/config.json` as valid JSON and invoke gtdo with that path:

  ```json
  {
    "dir": "$WORK/gtdo-home",
    "files": {
      "todo": "$WORK/gtdo-home/todo.txt",
      "done": "$WORK/gtdo-home/done.txt",
      "report": "$WORK/gtdo-home/report.txt"
    }
  }
  ```

  Keep todo.sh's shell configuration separate and unchanged. Update the comments so they compare todo.sh's bash config with gtdo's JSON config.

- [ ] **Step 10: Remove the TOML module and run the complete executable suite**

  Run: `go mod tidy`

  Expected: `github.com/BurntSushi/toml` disappears from both `go.mod` and `go.sum`; no new dependency is added.

  Run: `gofmt -w internal/cli/list.go internal/cli/multiline_test.go internal/cli/preparse_test.go internal/cli/script_test.go internal/ui/color.go internal/ui/color_test.go`

  Run: `go test ./... -count=1`

  Expected: PASS across config, CLI testscript, todo, UI, and man packages.

  Run: `bash -n scripts/parity.sh`

  Expected: no output and exit 0.

- [ ] **Step 11: Commit the atomic format switch**

  ```bash
  git add go.mod go.sum internal/config internal/cli internal/ui scripts/parity.sh
  git commit -m "feat(config): replace toml with strict json"
  ```

### Task 2: Publish the live JSON configuration contract

**Files:**
- Modify: `README.md:52-70`
- Modify: `AGENTS.md:17-25`
- Modify: `CHANGELOG.md:7-16`
- Modify: `man/man_test.go:57-80`
- Modify: `man/gtdo.1.tmpl:142-159`
- Modify: `man/gtdo.1`

**Interfaces:**
- Consumes: the discovery order and JSON schema implemented in Task 1.
- Produces: current user/developer documentation and a generated man page that name only the supported JSON format while preserving the historical 0.1.0 changelog entry.

- [ ] **Step 1: Add a failing man-page contract test**

  Add this focused test to `man/man_test.go`:

  ```go
  func TestManPageDocumentsJSONConfig(t *testing.T) {
      data, err := os.ReadFile("gtdo.1")
      if err != nil {
          t.Fatal(err)
      }
      page := string(data)
      for _, want := range []string{"~/.config/gtdo/config.json", "/etc/gtdo/config.json"} {
          if !strings.Contains(page, want) {
              t.Errorf("man/gtdo.1 is missing %q", want)
          }
      }
      if strings.Contains(page, "config.toml") {
          t.Error("man/gtdo.1 still documents config.toml")
      }
  }
  ```

- [ ] **Step 2: Run the focused man test and verify it fails**

  Run: `go test ./man -run TestManPageDocumentsJSONConfig -count=1`

  Expected: FAIL because the committed man page still names `config.toml`.

- [ ] **Step 3: Replace the README example with the approved JSON schema**

  Document `~/.config/gtdo/config.json`, `-d`, `$GTDO_CONFIG`, the hard switch, and environment compatibility. Use a minimal valid example:

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
      "preserveLineNumbers": true
    }
  }
  ```

  State that unknown keys and `null` are errors and that omitted settings retain defaults. Do not document TOML fallback or aliases.

- [ ] **Step 4: Update developer guidance and changelog**

  In `AGENTS.md`, change the config test description from TOML to strict JSON and call out `encoding/json` plus camelCase schema expectations.

  Under `[Unreleased]` in `CHANGELOG.md`, add:

  ```markdown
  ### Changed

  - Replaced TOML configuration with a strict camelCase JSON format at
    `~/.config/gtdo/config.json`.
  ```

  Keep the 0.1.0 `TOML configuration` bullet unchanged because it accurately describes that released version.

- [ ] **Step 5: Update the man template and regenerate the committed page**

  Change the FILES entry in `man/gtdo.1.tmpl` to:

  ```roff
  .I ~/.config/gtdo/config.json
  User configuration, searched after the
  .B \-d
  option and
  .BR $GTDO_CONFIG ,
  before
  .IR /etc/gtdo/config.json .
  ```

  Run: `make man`

  Expected: `man/gtdo.1` is regenerated with the JSON paths.

- [ ] **Step 6: Verify live documentation tests**

  Run: `go test ./man -count=1`

  Expected: PASS, including byte-for-byte regeneration and JSON config assertions.

  Run: `rg -n -i 'toml|config\.toml' README.md AGENTS.md man/gtdo.1.tmpl man/gtdo.1`

  Expected: no matches.

- [ ] **Step 7: Commit live documentation**

  ```bash
  git add README.md AGENTS.md CHANGELOG.md man/man_test.go man/gtdo.1.tmpl man/gtdo.1
  git commit -m "docs: publish json configuration contract"
  ```

### Task 3: Align normative specs and plans, then verify the repository

**Files:**
- Modify: `docs/superpowers/specs/2026-08-07-gtdo-migracion-todotxt-cli-design.md`
- Modify: `docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.en.md`
- Modify: `docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.md`
- Modify: `docs/superpowers/specs/2026-08-10-allowed-contexts-projects-design.md`
- Modify: `docs/superpowers/plans/2026-08-10-allowed-contexts-projects.md`
- Modify: `docs/superpowers/specs/2026-08-10-task-format-design.md`

**Interfaces:**
- Consumes: the approved JSON design and implemented schema from Task 1.
- Produces: normative future-work documents that reference `json.go`, `behaviour`, camelCase keys, JSON arrays, and `encoding/json` consistently.

- [ ] **Step 1: Update the original migration design and bilingual plans**

  Replace the format, dependency, precedence, search-path, schema example, testing, and acceptance references. The canonical schema language must be:

  ```markdown
  Config search order: `-d PATH` / `$GTDO_CONFIG` >
  `~/.config/gtdo/config.json` > `/etc/gtdo/config.json`.

  Precedence: CLI flags > environment variables > JSON > defaults.
  ```

  Convert schema code fences to `json` and use the approved `dir`, `files`, `behaviour`, and `colors` shape. Replace BurntSushi references with standard-library `encoding/json`, and replace `internal/config/toml.go` task references with `internal/config/json.go` plus `internal/config/loader.go`.

- [ ] **Step 2: Update the allowed-context/project spec and implementation plan**

  Preserve that feature's nil-versus-empty list semantics while translating its on-disk shape to:

  ```json
  {
    "behaviour": {
      "allowedContexts": ["@home", "@work"],
      "allowedProjects": ["+gtdo"]
    }
  }
  ```

  Its plan must target `behaviourJSON.AllowedContexts []string` with `json:"allowedContexts"` and `behaviourJSON.AllowedProjects []string` with `json:"allowedProjects"`, modify `internal/config/json.go`, and run JSON-named tests. Remove TOML-only wording and the BurntSushi dependency from its header.

- [ ] **Step 3: Update the task-format design**

  Translate its configuration example and contract to:

  ```json
  {
    "behaviour": {
      "taskFormat": "{priority} {creationDate} {description}"
    }
  }
  ```

  Replace `task_format`, `[behavior]`, TOML schema, and TOML-loaded test wording with `taskFormat`, `behaviour`, JSON schema, and JSON-loaded wording. Preserve the feature's task-format semantics unchanged.

- [ ] **Step 4: Scan normative documents for stale format instructions**

  Run:

  ```bash
  rg -n -i 'toml|config\.toml|BurntSushi|\[behavior\]|\[paths\]|\[colors\]' \
    docs/superpowers/specs/2026-08-07-gtdo-migracion-todotxt-cli-design.md \
    docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.en.md \
    docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.md \
    docs/superpowers/specs/2026-08-10-allowed-contexts-projects-design.md \
    docs/superpowers/plans/2026-08-10-allowed-contexts-projects.md \
    docs/superpowers/specs/2026-08-10-task-format-design.md
  ```

  Expected: no matches. Do not rewrite `docs/superpowers/specs/2026-08-10-json-config-design.md` or this implementation plan merely because they describe removal of TOML.

- [ ] **Step 5: Commit the normative documentation migration**

  ```bash
  git add docs/superpowers/specs/2026-08-07-gtdo-migracion-todotxt-cli-design.md \
    docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.en.md \
    docs/superpowers/plans/2026-08-07-gtdo-migracion-todotxt-cli.md \
    docs/superpowers/specs/2026-08-10-allowed-contexts-projects-design.md \
    docs/superpowers/plans/2026-08-10-allowed-contexts-projects.md \
    docs/superpowers/specs/2026-08-10-task-format-design.md
  git commit -m "docs: align plans with json configuration"
  ```

- [ ] **Step 6: Run final formatting, build, tests, vet, man, and stale-reference checks**

  Run: `gofmt -w internal/config/*.go internal/cli/list.go internal/cli/multiline_test.go internal/cli/preparse_test.go internal/cli/script_test.go internal/ui/color.go internal/ui/color_test.go man/man_test.go`

  Run: `make build`

  Expected: `./gtdo` builds successfully with version metadata.

  Run: `go test ./... -count=1`

  Expected: PASS.

  Run: `go vet ./internal/config/`

  Expected: no output and exit 0.

  Run: `make man`

  Run: `git diff --exit-code -- man/gtdo.1`

  Expected: the generated man page remains byte-for-byte committed.

  Run: `go list -m all`

  Expected: output does not contain `github.com/BurntSushi/toml`.

  Run: `rg -n -i 'toml|config\.toml|BurntSushi' --glob '!CHANGELOG.md' --glob '!docs/superpowers/specs/2026-08-10-json-config-design.md' --glob '!docs/superpowers/plans/2026-08-10-json-configuration.md'`

  Expected: no matches. The excluded changelog contains only the accurate historical 0.1.0 TOML entry; the two excluded migration documents intentionally describe the removed format.

  Run: `git status --short`

  Expected: no output.
