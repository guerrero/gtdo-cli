package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// The on-disk TOML schema (§5.2). Every section decodes over its todo.sh
// defaults, so BurntSushi/toml touches only the keys the file actually sets.

// pathsTOML is the [paths] section.
type pathsTOML struct {
	Dir        string `toml:"dir"`
	TodoFile   string `toml:"todo_file"`
	DoneFile   string `toml:"done_file"`
	ReportFile string `toml:"report_file"`
}

// behaviorTOML is the [behavior] section. There is deliberately no plain
// key: §5.2 reserves plain for the CLI flags and TODOTXT_PLAIN.
type behaviorTOML struct {
	Force               bool   `toml:"force"`
	PreserveLineNumbers bool   `toml:"preserve_line_numbers"`
	AutoArchive         bool   `toml:"auto_archive"`
	DateOnAdd           bool   `toml:"date_on_add"`
	PriorityOnAdd       string `toml:"priority_on_add"`
	Verbose             int    `toml:"verbose"`
	DefaultAction       string `toml:"default_action"`
	SourceVar           string `toml:"sourcevar"`
	SentenceDelimiters  string `toml:"sentence_delimiters"`
}

// colorsTOML is the [colors] section plus its [colors.map] sub-table.
type colorsTOML struct {
	PriA string `toml:"pri_a"`
	PriB string `toml:"pri_b"`
	PriC string `toml:"pri_c"`
	PriD string `toml:"pri_d"`
	PriE string `toml:"pri_e"`
	PriF string `toml:"pri_f"`
	PriG string `toml:"pri_g"`
	PriH string `toml:"pri_h"`
	PriI string `toml:"pri_i"`
	PriJ string `toml:"pri_j"`
	PriK string `toml:"pri_k"`
	PriL string `toml:"pri_l"`
	PriM string `toml:"pri_m"`
	PriN string `toml:"pri_n"`
	PriO string `toml:"pri_o"`
	PriP string `toml:"pri_p"`
	PriQ string `toml:"pri_q"`
	PriR string `toml:"pri_r"`
	PriS string `toml:"pri_s"`
	PriT string `toml:"pri_t"`
	PriU string `toml:"pri_u"`
	PriV string `toml:"pri_v"`
	PriW string `toml:"pri_w"`
	PriX string `toml:"pri_x"`
	PriY string `toml:"pri_y"`
	PriZ string `toml:"pri_z"`

	ColorDone    string `toml:"color_done"`
	ColorProject string `toml:"color_project"`
	ColorContext string `toml:"color_context"`
	ColorDate    string `toml:"color_date"`
	ColorNumber  string `toml:"color_number"`
	ColorMeta    string `toml:"color_meta"`

	// Map is the [colors.map] table: name → ANSI code overrides for the
	// built-in color names.
	Map map[string]string `toml:"map"`
}

// fileConfig is the whole TOML file, pre-filled with the defaults.
type fileConfig struct {
	Paths    pathsTOML    `toml:"paths"`
	Behavior behaviorTOML `toml:"behavior"`
	Colors   colorsTOML   `toml:"colors"`
}

// systemConfigPath is the last candidate in the §5.1 search order.
const systemConfigPath = "/etc/gtdo/config.toml"

// findConfigFile returns the first existing file in §5.1 order: -d PATH,
// $GTDO_CONFIG, ~/.config/gtdo/config.toml, then the system path. It returns
// "" when none exists and load falls back to the defaults. A -d target that
// does not exist is skipped, not an error.
func findConfigFile(opts Options, home, systemPath string) string {
	candidates := []string{opts.ConfigPath}
	if v, ok := envString("GTDO_CONFIG"); ok {
		candidates = append(candidates, expandHome(v, home))
	}
	candidates = append(candidates,
		filepath.Join(home, ".config", "gtdo", "config.toml"),
		systemPath,
	)
	for _, p := range candidates {
		if p != "" && exists(p) {
			return p
		}
	}
	return ""
}

// load resolves the configuration from opts, the environment, and the first
// existing config file under home/systemPath.
func load(opts Options, home, systemPath string) (Config, error) {
	path := findConfigFile(opts, home, systemPath)
	f := defaultFileConfig()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("config: read %s: %w", path, err)
		}
		if _, err := toml.Decode(string(data), &f); err != nil {
			return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}
	cfg := resolve(opts, f, home)
	cfg.ConfigPath = path
	return cfg, nil
}
