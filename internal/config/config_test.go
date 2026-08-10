package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// home returns a fake $HOME for a test.
func home(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// writeConfig writes body to path (creating parent directories) and returns
// the path.
func writeConfig(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// withOpts returns opts plus a ConfigPath pointing at a file with body.
func withOpts(t *testing.T, opts Options, body string) Options {
	t.Helper()
	opts.ConfigPath = writeConfig(t, filepath.Join(t.TempDir(), "config.json"), body)
	return opts
}

// loadAt loads with an isolated HOME and a chosen system config path.
func loadAt(t *testing.T, opts Options, h, sys string) Config {
	t.Helper()
	cfg, err := load(opts, h, sys)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// loadWith loads with an isolated HOME and a nonexistent system config.
func loadWith(t *testing.T, opts Options, h string) Config {
	t.Helper()
	return loadAt(t, opts, h, filepath.Join(t.TempDir(), "nonexistent", "config.json"))
}

// TestDefaults pins the todo.sh defaults (§5.3) with no config file, no env,
// and no flags.
func TestDefaults(t *testing.T) {
	h := home(t)
	cfg := loadWith(t, Options{}, h)

	if cfg.ConfigPath != "" {
		t.Errorf("ConfigPath = %q, want empty (no file found)", cfg.ConfigPath)
	}
	if cfg.Dir != h {
		t.Errorf("Dir = %q, want %q", cfg.Dir, h)
	}
	if cfg.TodoFile != filepath.Join(h, "todo.txt") {
		t.Errorf("TodoFile = %q, want %q", cfg.TodoFile, filepath.Join(h, "todo.txt"))
	}
	if cfg.DoneFile != filepath.Join(h, "done.txt") {
		t.Errorf("DoneFile = %q, want %q", cfg.DoneFile, filepath.Join(h, "done.txt"))
	}
	if cfg.ReportFile != filepath.Join(h, "report.txt") {
		t.Errorf("ReportFile = %q, want %q", cfg.ReportFile, filepath.Join(h, "report.txt"))
	}
	if cfg.Force {
		t.Error("Force = true, want false")
	}
	if !cfg.PreserveLineNumbers {
		t.Error("PreserveLineNumbers = false, want true")
	}
	if !cfg.AutoArchive {
		t.Error("AutoArchive = false, want true")
	}
	if cfg.DateOnAdd {
		t.Error("DateOnAdd = true, want false")
	}
	if cfg.PriorityOnAdd != "" {
		t.Errorf("PriorityOnAdd = %q, want empty", cfg.PriorityOnAdd)
	}
	if cfg.Verbose != 1 {
		t.Errorf("Verbose = %d, want 1", cfg.Verbose)
	}
	if cfg.DefaultAction != "" {
		t.Errorf("DefaultAction = %q, want empty", cfg.DefaultAction)
	}
	if cfg.SourceVar != "" {
		t.Errorf("SourceVar = %q, want empty", cfg.SourceVar)
	}
	if cfg.SentenceDelimiters != ",.:;" {
		t.Errorf("SentenceDelimiters = %q, want %q", cfg.SentenceDelimiters, ",.:;")
	}
	if cfg.TaskFormat != DefaultTaskFormat {
		t.Errorf("TaskFormat = %q, want %q", cfg.TaskFormat, DefaultTaskFormat)
	}
	if cfg.Plain {
		t.Error("Plain = true, want false")
	}
}

func TestTaskFormatFromTOML(t *testing.T) {
	h := home(t)
	body := "[behavior]\ntask_format = \"[project][content][keywords][context]\"\n"
	cfg := loadWith(t, withOpts(t, Options{}, body), h)
	want := "[project][content][keywords][context]"
	if cfg.TaskFormat != want {
		t.Errorf("TaskFormat = %q, want %q", cfg.TaskFormat, want)
	}
}

// TestFileSearchOrder pins §5.1: -d PATH, $GTDO_CONFIG,
// ~/.config/gtdo/config.json, /etc/gtdo/config.json; none existing → defaults.
// Each subtest gets a fresh home so earlier subtests' files never leak in.
func TestFileSearchOrder(t *testing.T) {
	t.Run("dash d wins", func(t *testing.T) {
		h := home(t)
		d := writeConfig(t, filepath.Join(t.TempDir(), "d", "config.json"), `{"behaviour":{"verbose":2}}`)
		env := writeConfig(t, filepath.Join(t.TempDir(), "env", "config.json"), `{"behaviour":{"verbose":3}}`)
		writeConfig(t, filepath.Join(h, ".config", "gtdo", "config.json"), `{"behaviour":{"verbose":4}}`)
		t.Setenv("GTDO_CONFIG", env)
		cfg := loadWith(t, Options{ConfigPath: d}, h)
		if cfg.Verbose != 2 || cfg.ConfigPath != d {
			t.Errorf("Verbose = %d, ConfigPath = %q; want 2, %q", cfg.Verbose, cfg.ConfigPath, d)
		}
	})

	t.Run("gtdo config env wins over home", func(t *testing.T) {
		h := home(t)
		env := writeConfig(t, filepath.Join(t.TempDir(), "env", "config.json"), `{"behaviour":{"verbose":3}}`)
		writeConfig(t, filepath.Join(h, ".config", "gtdo", "config.json"), `{"behaviour":{"verbose":4}}`)
		t.Setenv("GTDO_CONFIG", env)
		cfg := loadWith(t, Options{}, h)
		if cfg.Verbose != 3 || cfg.ConfigPath != env {
			t.Errorf("Verbose = %d, ConfigPath = %q; want 3, %q", cfg.Verbose, cfg.ConfigPath, env)
		}
	})

	t.Run("home config wins over system", func(t *testing.T) {
		h := home(t)
		sys := writeConfig(t, filepath.Join(t.TempDir(), "etc", "gtdo", "config.json"), `{"behaviour":{"verbose":5}}`)
		homeCfg := writeConfig(t, filepath.Join(h, ".config", "gtdo", "config.json"), `{"behaviour":{"verbose":4}}`)
		cfg := loadAt(t, Options{}, h, sys)
		if cfg.Verbose != 4 || cfg.ConfigPath != homeCfg {
			t.Errorf("Verbose = %d, ConfigPath = %q; want 4, %q", cfg.Verbose, cfg.ConfigPath, homeCfg)
		}
	})

	t.Run("system config is the last resort", func(t *testing.T) {
		h := home(t)
		sys := writeConfig(t, filepath.Join(t.TempDir(), "etc", "gtdo", "config.json"), `{"behaviour":{"verbose":5}}`)
		cfg := loadAt(t, Options{}, h, sys)
		if cfg.Verbose != 5 || cfg.ConfigPath != sys {
			t.Errorf("Verbose = %d, ConfigPath = %q; want 5, %q", cfg.Verbose, cfg.ConfigPath, sys)
		}
	})

	t.Run("missing dash d falls through", func(t *testing.T) {
		h := home(t)
		env := writeConfig(t, filepath.Join(t.TempDir(), "env", "config.json"), `{"behaviour":{"verbose":3}}`)
		t.Setenv("GTDO_CONFIG", env)
		cfg := loadWith(t, Options{ConfigPath: filepath.Join(t.TempDir(), "missing.json")}, h)
		if cfg.Verbose != 3 || cfg.ConfigPath != env {
			t.Errorf("Verbose = %d, ConfigPath = %q; want 3, %q", cfg.Verbose, cfg.ConfigPath, env)
		}
	})

	t.Run("gtdo config expands home", func(t *testing.T) {
		h := home(t)
		cfgFile := writeConfig(t, filepath.Join(h, "cfg.json"), `{"behaviour":{"verbose":6}}`)
		t.Setenv("GTDO_CONFIG", "~/cfg.json")
		cfg := loadWith(t, Options{}, h)
		if cfg.Verbose != 6 || cfg.ConfigPath != cfgFile {
			t.Errorf("Verbose = %d, ConfigPath = %q; want 6, %q", cfg.Verbose, cfg.ConfigPath, cfgFile)
		}
	})

	t.Run("no file anywhere means defaults", func(t *testing.T) {
		h := home(t)
		t.Setenv("GTDO_CONFIG", "")
		cfg := loadWith(t, Options{}, h)
		if cfg.ConfigPath != "" || cfg.Verbose != 1 {
			t.Errorf("ConfigPath = %q, Verbose = %d; want empty, 1", cfg.ConfigPath, cfg.Verbose)
		}
	})

	t.Run("legacy home toml is ignored", func(t *testing.T) {
		h := home(t)
		writeConfig(t, filepath.Join(h, ".config", "gtdo", "config.toml"), `{"behaviour":{"verbose":9}}`)
		cfg := loadWith(t, Options{}, h)
		if cfg.ConfigPath != "" || cfg.Verbose != 1 {
			t.Fatalf("ConfigPath = %q, Verbose = %d; want empty, 1", cfg.ConfigPath, cfg.Verbose)
		}
	})
}

// TestPrecedenceBools stacks JSON, env, and flag layers for each boolean key:
// the highest layer that sets the value wins (§5.3).
func TestPrecedenceBools(t *testing.T) {
	cases := []struct {
		name    string
		jsonKey string // "" when the key has no JSON counterpart
		envName string
		mkFlag  func(bool) Options
		get     func(Config) bool
		def     bool
	}{
		{"force", "force", "TODOTXT_FORCE", func(b bool) Options { return Options{Force: b, ForceSet: true} }, func(c Config) bool { return c.Force }, false},
		{"preserveLineNumbers", "preserveLineNumbers", "TODOTXT_PRESERVE_LINE_NUMBERS", func(b bool) Options { return Options{Preserve: b, PreserveSet: true} }, func(c Config) bool { return c.PreserveLineNumbers }, true},
		{"autoArchive", "autoArchive", "TODOTXT_AUTO_ARCHIVE", func(b bool) Options { return Options{AutoArchive: b, AutoArchiveSet: true} }, func(c Config) bool { return c.AutoArchive }, true},
		{"dateOnAdd", "dateOnAdd", "TODOTXT_DATE_ON_ADD", func(b bool) Options { return Options{DateOnAdd: b, DateOnAddSet: true} }, func(c Config) bool { return c.DateOnAdd }, false},
		{"plain", "", "TODOTXT_PLAIN", func(b bool) Options { return Options{Plain: b, PlainSet: true} }, func(c Config) bool { return c.Plain }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := home(t)
			body := `{}`
			if tc.jsonKey != "" {
				body = fmt.Sprintf(`{"behaviour":{"%s":true}}`, tc.jsonKey)
			}

			// Default layer.
			if got := tc.get(loadWith(t, Options{}, h)); got != tc.def {
				t.Errorf("default = %v, want %v", got, tc.def)
			}

			// JSON layer beats the default.
			if tc.jsonKey != "" {
				if got := tc.get(loadWith(t, withOpts(t, Options{}, body), h)); !got {
					t.Error("JSON layer did not win over the default")
				}
			}

			// Env layer beats JSON.
			t.Setenv(tc.envName, "0")
			if got := tc.get(loadWith(t, withOpts(t, Options{}, body), h)); got {
				t.Error("env layer did not win over JSON")
			}

			// Flag layer beats env.
			if got := tc.get(loadWith(t, withOpts(t, tc.mkFlag(true), body), h)); !got {
				t.Error("flag layer did not win over env")
			}
		})
	}
}

// TestPrecedenceStrings stacks JSON and env layers for each string key.
func TestPrecedenceStrings(t *testing.T) {
	cases := []struct {
		name    string
		jsonKey string
		envName string
		get     func(Config) string
		def     string
	}{
		{"priorityOnAdd", "priorityOnAdd", "TODOTXT_PRIORITY_ON_ADD", func(c Config) string { return c.PriorityOnAdd }, ""},
		{"defaultAction", "defaultAction", "TODOTXT_DEFAULT_ACTION", func(c Config) string { return c.DefaultAction }, ""},
		{"sourceVar", "sourceVar", "TODOTXT_SOURCEVAR", func(c Config) string { return c.SourceVar }, ""},
		{"sentenceDelimiters", "sentenceDelimiters", "SENTENCE_DELIMITERS", func(c Config) string { return c.SentenceDelimiters }, ",.:;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := home(t)
			body := fmt.Sprintf(`{"behaviour":{"%s":"json"}}`, tc.jsonKey)

			// Default layer.
			if got := tc.get(loadWith(t, Options{}, h)); got != tc.def {
				t.Errorf("default = %q, want %q", got, tc.def)
			}

			// JSON layer beats the default.
			if got := tc.get(loadWith(t, withOpts(t, Options{}, body), h)); got != "json" {
				t.Errorf("JSON = %q, want %q", got, "json")
			}

			// Env layer beats JSON.
			t.Setenv(tc.envName, "env")
			if got := tc.get(loadWith(t, withOpts(t, Options{}, body), h)); got != "env" {
				t.Errorf("env = %q, want %q", got, "env")
			}
		})
	}
}

// TestVerboseCounting pins the §5.3 -v rule: TODOTXT_VERBOSE wins when it is
// defined; otherwise the -v count wins over JSON; otherwise JSON; otherwise 1.
func TestVerboseCounting(t *testing.T) {
	h := home(t)
	body := `{"behaviour":{"verbose":3}}`

	cases := []struct {
		name    string
		env     string
		envSet  bool
		count   int
		useJSON bool
		want    int
	}{
		{"no env, no flags, no json", "", false, 0, false, 1},
		{"no env, no flags, json", "", false, 0, true, 3},
		{"two dash v beats json", "", false, 2, true, 2},
		{"env beats dash v", "5", true, 3, true, 5},
		{"env zero suppresses output", "0", true, 3, true, 0},
		{"unparseable env is unset", "abc", true, 2, true, 2},
		{"empty env is unset", "", true, 0, true, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := Options{VerboseCount: tc.count}
			if tc.useJSON {
				opts = withOpts(t, opts, body)
			}
			if tc.envSet {
				t.Setenv("TODOTXT_VERBOSE", tc.env)
			}
			if got := loadWith(t, opts, h).Verbose; got != tc.want {
				t.Errorf("Verbose = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestEnvParsing covers the bool/int/string env parsing: empty means unset,
// only "0"/"false"/"no"/"off" disable a bool, and a garbage int is ignored.
func TestEnvParsing(t *testing.T) {
	h := home(t)

	t.Run("bools", func(t *testing.T) {
		cases := []struct {
			value string
			want  bool
		}{
			{"1", true},
			{"true", true},
			{"TRUE", true},
			{"yes", true},
			{"on", true},
			{"0", false},
			{"false", false},
			{"no", false},
			{"off", false},
		}
		for _, tc := range cases {
			t.Setenv("TODOTXT_FORCE", tc.value)
			if got := loadWith(t, Options{}, h).Force; got != tc.want {
				t.Errorf("TODOTXT_FORCE=%q: Force = %v, want %v", tc.value, got, tc.want)
			}
		}
	})

	t.Run("empty bool env is unset", func(t *testing.T) {
		t.Setenv("TODOTXT_FORCE", "")
		if got := loadWith(t, withOpts(t, Options{}, `{"behaviour":{"force":true}}`), h).Force; !got {
			t.Error("empty TODOTXT_FORCE shadowed the JSON value")
		}
	})

	t.Run("int", func(t *testing.T) {
		t.Setenv("TODOTXT_VERBOSE", "2")
		if got := loadWith(t, Options{}, h).Verbose; got != 2 {
			t.Errorf("Verbose = %d, want 2", got)
		}
	})

	t.Run("garbage int is unset", func(t *testing.T) {
		t.Setenv("TODOTXT_VERBOSE", "abc")
		if got := loadWith(t, withOpts(t, Options{}, `{"behaviour":{"verbose":3}}`), h).Verbose; got != 3 {
			t.Errorf("Verbose = %d, want JSON value 3", got)
		}
	})

	t.Run("string passthrough", func(t *testing.T) {
		t.Setenv("TODO_FILE", "custom.txt")
		if got := loadWith(t, Options{}, h).TodoFile; got != "custom.txt" {
			t.Errorf("TodoFile = %q, want %q", got, "custom.txt")
		}
	})
}

// TestPathExpansion covers "~" and "$HOME" expansion in JSON and env paths,
// and the derived default file locations under the resolved dir.
func TestPathExpansion(t *testing.T) {
	h := home(t)

	t.Run("json paths", func(t *testing.T) {
		cfg := loadWith(t, withOpts(t, Options{}, `{"dir":"~/todo","files":{"todo":"~","done":"$HOME/done.txt","report":"a$HOME/b"}}`), h)
		if cfg.Dir != filepath.Join(h, "todo") {
			t.Errorf("Dir = %q, want %q", cfg.Dir, filepath.Join(h, "todo"))
		}
		if cfg.TodoFile != h {
			t.Errorf("TodoFile = %q, want %q", cfg.TodoFile, h)
		}
		if cfg.DoneFile != filepath.Join(h, "done.txt") {
			t.Errorf("DoneFile = %q, want %q", cfg.DoneFile, filepath.Join(h, "done.txt"))
		}
		if got := cfg.ReportFile; got != "a$HOME/b" {
			t.Errorf("ReportFile = %q, want unexpanded %q", got, "a$HOME/b")
		}
	})

	t.Run("env paths", func(t *testing.T) {
		t.Setenv("TODO_DIR", "~/work")
		t.Setenv("TODO_FILE", "$HOME/x.txt")
		cfg := loadWith(t, Options{}, h)
		if cfg.Dir != filepath.Join(h, "work") {
			t.Errorf("Dir = %q, want %q", cfg.Dir, filepath.Join(h, "work"))
		}
		if cfg.TodoFile != filepath.Join(h, "x.txt") {
			t.Errorf("TodoFile = %q, want %q", cfg.TodoFile, filepath.Join(h, "x.txt"))
		}
	})

	t.Run("tilde user is not expanded", func(t *testing.T) {
		t.Setenv("TODO_DIR", "~other/todo")
		if got := loadWith(t, Options{}, h).Dir; got != "~other/todo" {
			t.Errorf("Dir = %q, want literal %q", got, "~other/todo")
		}
	})

	t.Run("file defaults follow resolved dir", func(t *testing.T) {
		cfg := loadWith(t, withOpts(t, Options{}, `{"dir":"~/todo"}`), h)
		for name, got := range map[string]string{
			"TodoFile":   cfg.TodoFile,
			"DoneFile":   cfg.DoneFile,
			"ReportFile": cfg.ReportFile,
		} {
			want := filepath.Join(h, "todo", strings.ToLower(strings.TrimSuffix(name, "File"))+".txt")
			if got != want {
				t.Errorf("%s = %q, want %q", name, got, want)
			}
		}
	})
}

// TestJSONFullSchema decodes every key of the §5.2 schema at once.
func TestJSONFullSchema(t *testing.T) {
	h := home(t)
	body := `{"dir":"~/todo","files":{"todo":"~/todo/todo.txt","done":"~/todo/done.txt","report":"~/todo/report.txt"},"behaviour":{"force":true,"preserveLineNumbers":false,"autoArchive":false,"dateOnAdd":true,"priorityOnAdd":"B","verbose":2,"defaultAction":"list","sourceVar":"~/done.txt","sentenceDelimiters":".;"},"colors":{"priA":"yellow","priB":"\\033[0;32m","colorProject":"light_cyan","colorMeta":"","map":{"yellow":"\\033[1;43m"}}}`
	path := writeConfig(t, filepath.Join(t.TempDir(), "config.json"), body)
	cfg := loadAt(t, Options{ConfigPath: path}, h, filepath.Join(t.TempDir(), "nonexistent", "config.json"))

	if cfg.ConfigPath != path {
		t.Errorf("ConfigPath = %q, want %q", cfg.ConfigPath, path)
	}
	if cfg.Dir != filepath.Join(h, "todo") {
		t.Errorf("Dir = %q, want %q", cfg.Dir, filepath.Join(h, "todo"))
	}
	if cfg.TodoFile != filepath.Join(h, "todo", "todo.txt") {
		t.Errorf("TodoFile = %q", cfg.TodoFile)
	}
	if cfg.DoneFile != filepath.Join(h, "todo", "done.txt") {
		t.Errorf("DoneFile = %q", cfg.DoneFile)
	}
	if cfg.ReportFile != filepath.Join(h, "todo", "report.txt") {
		t.Errorf("ReportFile = %q", cfg.ReportFile)
	}
	if !cfg.Force || cfg.PreserveLineNumbers || cfg.AutoArchive || !cfg.DateOnAdd {
		t.Errorf("bools = force:%v preserve:%v archive:%v date:%v, want true,false,false,true",
			cfg.Force, cfg.PreserveLineNumbers, cfg.AutoArchive, cfg.DateOnAdd)
	}
	if cfg.PriorityOnAdd != "B" || cfg.Verbose != 2 || cfg.DefaultAction != "list" || cfg.SourceVar != "~/done.txt" || cfg.SentenceDelimiters != ".;" {
		t.Errorf("strings = %q/%d/%q/%q/%q, want B/2/list/~/done.txt/.;",
			cfg.PriorityOnAdd, cfg.Verbose, cfg.DefaultAction, cfg.SourceVar, cfg.SentenceDelimiters)
	}

	if got := cfg.Color("pri_a"); got != "\x1b[1;43m" {
		t.Errorf("Color(pri_a) = %q, want overridden yellow %q", got, "\x1b[1;43m")
	}
	if got := cfg.Color("pri_b"); got != "\x1b[0;32m" {
		t.Errorf("Color(pri_b) = %q, want raw ANSI %q", got, "\x1b[0;32m")
	}
	if got := cfg.Color("color_project"); got != "\x1b[1;36m" {
		t.Errorf("Color(color_project) = %q, want %q", got, "\x1b[1;36m")
	}
	if got := cfg.Color("color_meta"); got != "" {
		t.Errorf("Color(color_meta) = %q, want empty (off)", got)
	}
	if got := cfg.Color("pri_c"); got != "\x1b[1;34m" {
		t.Errorf("Color(pri_c) = %q, want default %q", got, "\x1b[1;34m")
	}
	if got := cfg.Color("color_done"); got != "\x1b[0;37m" {
		t.Errorf("Color(color_done) = %q, want default %q", got, "\x1b[0;37m")
	}
}

// TestColorResolution covers colors values: map names (case-insensitive),
// raw ANSI strings with \033 translated, empty = off, and unknown names
// passed through verbatim.
func TestColorResolution(t *testing.T) {
	h := home(t)

	cases := []struct {
		name string
		body string
		role string
		want string
	}{
		{"map name", `{"colors":{"priA":"yellow"}}`, "pri_a", "\x1b[1;33m"},
		{"map name case-insensitive", `{"colors":{"priA":"YELLOW"}}`, "pri_a", "\x1b[1;33m"},
		{"raw ansi", `{"colors":{"priB":"\\033[0;32m"}}`, "pri_b", "\x1b[0;32m"},
		{"empty is off", `{"colors":{"priC":""}}`, "pri_c", ""},
		{"unknown name is raw", `{"colors":{"priD":"blink"}}`, "pri_d", "blink"},
		{"magenta aliases purple", `{"colors":{"priA":"magenta"}}`, "pri_a", "\x1b[0;35m"},
		{"map override", `{"colors":{"priA":"yellow","map":{"yellow":"\\033[1;43m"}}}`, "pri_a", "\x1b[1;43m"},
		{"map key case-insensitive", `{"colors":{"priA":"yellow","map":{"YELLOW":"\\033[1;43m"}}}`, "pri_a", "\x1b[1;43m"},
		{"map names stay resolvable after override", `{"colors":{"priA":"red","map":{"yellow":"\\033[1;43m"}}}`, "pri_a", "\x1b[0;31m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadWith(t, withOpts(t, Options{}, tc.body), h)
			if got := cfg.Color(tc.role); got != tc.want {
				t.Errorf("Color(%s) = %q, want %q", tc.role, got, tc.want)
			}
		})
	}
}

// TestPriorityColorFallback pins todo.sh's PRI_<letter> / PRI_X fallback.
func TestPriorityColorFallback(t *testing.T) {
	h := home(t)
	cfg := loadWith(t, Options{}, h)

	if got := cfg.PriorityColor('A'); got != "\x1b[1;33m" {
		t.Errorf("PriorityColor(A) = %q, want %q", got, "\x1b[1;33m")
	}
	if got := cfg.PriorityColor('B'); got != "\x1b[0;32m" {
		t.Errorf("PriorityColor(B) = %q, want %q", got, "\x1b[0;32m")
	}
	if got := cfg.PriorityColor('Z'); got != "\x1b[1;37m" {
		t.Errorf("PriorityColor(Z) = %q, want pri_x %q", got, "\x1b[1;37m")
	}

	cfg = loadWith(t, withOpts(t, Options{}, `{"colors":{"priD":"cyan"}}`), h)
	if got := cfg.PriorityColor('D'); got != "\x1b[0;36m" {
		t.Errorf("PriorityColor(D) = %q, want %q", got, "\x1b[0;36m")
	}
}

// TestPlainColors pins plain mode: every color resolves to "".
func TestPlainColors(t *testing.T) {
	h := home(t)
	body := `{"colors":{"priA":"yellow","colorProject":"red"}}`

	t.Run("flag", func(t *testing.T) {
		cfg := loadWith(t, withOpts(t, Options{Plain: true, PlainSet: true}, body), h)
		if !cfg.Plain {
			t.Fatal("Plain = false, want true")
		}
		for _, role := range []string{"pri_a", "pri_b", "color_done", "color_project", "color_context", "color_date", "color_number", "color_meta"} {
			if got := cfg.Color(role); got != "" {
				t.Errorf("Color(%s) = %q, want empty in plain mode", role, got)
			}
		}
		if got := cfg.PriorityColor('A'); got != "" {
			t.Errorf("PriorityColor(A) = %q, want empty in plain mode", got)
		}
	})

	t.Run("env", func(t *testing.T) {
		t.Setenv("TODOTXT_PLAIN", "1")
		cfg := loadWith(t, withOpts(t, Options{}, body), h)
		if !cfg.Plain {
			t.Fatal("Plain = false, want true")
		}
		if got := cfg.Color("pri_a"); got != "" {
			t.Errorf("Color(pri_a) = %q, want empty in plain mode", got)
		}
	})
}

// TestBuiltinColorMap pins the 16 ANSI color names plus NONE and DEFAULT to
// the exact codes todo.sh exports (todo.sh lines ~669-696).
func TestBuiltinColorMap(t *testing.T) {
	want := map[string]string{
		"none":         "",
		"black":        "\x1b[0;30m",
		"red":          "\x1b[0;31m",
		"green":        "\x1b[0;32m",
		"brown":        "\x1b[0;33m",
		"blue":         "\x1b[0;34m",
		"purple":       "\x1b[0;35m",
		"cyan":         "\x1b[0;36m",
		"light_grey":   "\x1b[0;37m",
		"dark_grey":    "\x1b[1;30m",
		"light_red":    "\x1b[1;31m",
		"light_green":  "\x1b[1;32m",
		"yellow":       "\x1b[1;33m",
		"light_blue":   "\x1b[1;34m",
		"light_purple": "\x1b[1;35m",
		"light_cyan":   "\x1b[1;36m",
		"white":        "\x1b[1;37m",
		"default":      "\x1b[0m",
		"magenta":      "\x1b[0;35m", // alias for purple (brief §5.2 wording)
	}
	got := builtinColorMap()
	for name, code := range want {
		if got[name] != code {
			t.Errorf("builtinColorMap[%s] = %q, want %q", name, got[name], code)
		}
	}
	if len(got) != len(want) {
		t.Errorf("builtinColorMap has %d entries, want %d", len(got), len(want))
	}
}

// TestInvalidJSON: a malformed file is an error, not silent defaults.
func TestInvalidJSON(t *testing.T) {
	h := home(t)
	opts := withOpts(t, Options{}, `{"dir":`)
	if _, err := load(opts, h, filepath.Join(t.TempDir(), "nonexistent", "config.json")); err == nil {
		t.Error("load accepted malformed JSON")
	}
}
