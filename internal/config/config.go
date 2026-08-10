// Package config resolves gtdo's configuration: CLI flags (pre-parsed),
// environment variables, TOML file, and defaults, in that order of
// precedence (§5.3 of the design plan).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options carries the global flag results produced by cli.Preparse.
type Options struct {
	ConfigPath     string
	Force          bool
	ForceSet       bool
	Plain          bool
	PlainSet       bool
	Preserve       bool
	PreserveSet    bool
	AutoArchive    bool
	AutoArchiveSet bool
	VerboseCount   int
	HideProjects   bool
	HideContexts   bool
	HidePriority   bool
	Version        bool
}

// Config is the fully resolved configuration the rest of gtdo consumes:
// flags, environment, TOML file, and defaults merged in that order. Paths
// have ~ and $HOME expanded.
type Config struct {
	// ConfigPath is the TOML file that was loaded, or "" when none existed.
	ConfigPath string

	// Paths (§5.2 [paths]). The file defaults derive from the resolved Dir,
	// mirroring todo.cfg's ${TODO_DIR}/todo.txt and friends.
	Dir        string
	TodoFile   string
	DoneFile   string
	ReportFile string

	// Behavior (§5.2 [behavior]).
	Force               bool
	PreserveLineNumbers bool
	AutoArchive         bool
	EnableUUID          bool
	PriorityOnAdd       string
	Verbose             int
	DefaultAction       string
	SourceVar           string
	SentenceDelimiters  string

	// Plain disables all color output (flag -p/-c, TODOTXT_PLAIN).
	Plain bool

	// HideProjects, HideContexts, and HidePriority implement the -+ -@ -P
	// flags: odd counts hide the sigils and the (X) priority label in list
	// output (§6.1). Flags only, like todo.sh's getopts toggles.
	HideProjects bool
	HideContexts bool
	HidePriority bool

	colors map[string]string // resolved ANSI codes by role; "" = off
}

// Color returns the ANSI escape sequence for a color role — the [colors]
// keys: pri_a..pri_z, color_done, color_project, color_context, color_date,
// color_number, color_meta. It returns "" when the role is unset, unknown,
// or plain mode is on. internal/ui consumes these codes.
func (c *Config) Color(role string) string {
	return c.colors[strings.ToLower(role)]
}

// PriorityColor returns the color for a priority letter, falling back to
// pri_x when the letter has no color of its own — todo.sh's awk rule
// (PRI_<letter>, else PRI_X).
func (c *Config) PriorityColor(letter byte) string {
	if clr := c.Color("pri_" + string(letter)); clr != "" {
		return clr
	}
	return c.Color("pri_x")
}

// Load resolves the effective configuration. The TOML file is searched in
// §5.1 order; a missing file is not an error and yields the defaults.
func Load(opts Options) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("config: locate home directory: %w", err)
	}
	return load(opts, home, systemConfigPath)
}

// resolve merges the layers for every setting: CLI flags (opts) beat the
// environment, which beats the TOML file, which beats todo.sh's defaults.
func resolve(opts Options, f fileConfig, home string) Config {
	dir := firstNonEmpty(envOr("TODO_DIR", ""), f.Paths.Dir, home)
	dir = expandHome(dir, home)

	cfg := Config{
		Dir:                 dir,
		TodoFile:            expandHome(firstNonEmpty(envOr("TODO_FILE", ""), f.Paths.TodoFile, filepath.Join(dir, "todo.txt")), home),
		DoneFile:            expandHome(firstNonEmpty(envOr("DONE_FILE", ""), f.Paths.DoneFile, filepath.Join(dir, "done.txt")), home),
		ReportFile:          expandHome(firstNonEmpty(envOr("REPORT_FILE", ""), f.Paths.ReportFile, filepath.Join(dir, "report.txt")), home),
		Force:               pickBool(opts.ForceSet, opts.Force, "TODOTXT_FORCE", f.Behavior.Force),
		PreserveLineNumbers: pickBool(opts.PreserveSet, opts.Preserve, "TODOTXT_PRESERVE_LINE_NUMBERS", f.Behavior.PreserveLineNumbers),
		AutoArchive:         pickBool(opts.AutoArchiveSet, opts.AutoArchive, "TODOTXT_AUTO_ARCHIVE", f.Behavior.AutoArchive),
		EnableUUID:          pickBool(false, false, "GTDO_ENABLE_UUID", f.Behavior.EnableUUID),
		Plain:               pickBool(opts.PlainSet, opts.Plain, "TODOTXT_PLAIN", false),
		HideProjects:        opts.HideProjects,
		HideContexts:        opts.HideContexts,
		HidePriority:        opts.HidePriority,
		PriorityOnAdd:       pickString("TODOTXT_PRIORITY_ON_ADD", f.Behavior.PriorityOnAdd),
		DefaultAction:       pickString("TODOTXT_DEFAULT_ACTION", f.Behavior.DefaultAction),
		SourceVar:           pickString("TODOTXT_SOURCEVAR", f.Behavior.SourceVar),
		SentenceDelimiters:  pickString("SENTENCE_DELIMITERS", f.Behavior.SentenceDelimiters),
	}
	// The -v rule (§5.3): TODOTXT_VERBOSE wins when it is defined; otherwise
	// max(1, -v count) wins over the TOML value, which defaults to 1.
	cfg.Verbose = f.Behavior.Verbose
	if v, ok := envInt("TODOTXT_VERBOSE"); ok {
		cfg.Verbose = v
	} else if opts.VerboseCount > 0 {
		cfg.Verbose = opts.VerboseCount
	}

	cfg.colors = resolveColors(f.Colors, cfg.Plain)
	return cfg
}
