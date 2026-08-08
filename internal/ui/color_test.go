package ui

// The palette snapshot (Task 5, §5.2 colors / §6.2.4): FromConfig resolves
// todo.cfg's color assignments through internal/config — the single source
// of truth for the 16-name map — and the listing pipeline's colorizer is
// exercised with the palette. Expected outputs mirror the real todo.sh
// (tests/t1330-ls-highlighting.sh, t1360-ls-project-context-highlighting.sh,
// t1380-ls-date-number-metadata-highlighting.sh).

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/todo"
)

// compile-time check: the palette satisfies the pipeline's color hook.
var _ todo.Colorer = Color{}

// loadConfig resolves a config from a TOML body in an isolated temp dir.
// Colors come only from the TOML (§5.2: color env vars are not supported),
// so TODOTXT_PLAIN is pinned off for determinism unless a test set it first.
func loadConfig(t *testing.T, body string) config.Config {
	t.Helper()
	if _, ok := os.LookupEnv("TODOTXT_PLAIN"); !ok {
		t.Setenv("TODOTXT_PLAIN", "")
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(config.Options{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// formatLines runs the pipeline on consecutive tasks and returns the lines.
func formatLines(t *testing.T, texts []string, pal Color) []string {
	t.Helper()
	tasks := make([]todo.Task, len(texts))
	for i, text := range texts {
		tasks[i] = todo.Task{LineNumber: i + 1, Text: text}
	}
	lines, _, _, err := todo.Format(tasks, nil, todo.FormatOptions{Colors: pal})
	if err != nil {
		t.Fatal(err)
	}
	return lines
}

// TestFromConfigDefaults pins the todo.cfg palette (§5.2 defaults): pri_a..c
// and pri_x resolve to yellow/green/light_blue/white and color_done to
// light_grey; the word colors are off.
func TestFromConfigDefaults(t *testing.T) {
	pal := FromConfig(loadConfig(t, ""))
	want := Color{
		PriA:    "\x1b[1;33m",
		PriB:    "\x1b[0;32m",
		PriC:    "\x1b[1;34m",
		PriX:    "\x1b[1;37m",
		Done:    "\x1b[0;37m",
		Default: "\x1b[0m",
	}
	if !reflect.DeepEqual(pal, want) {
		t.Errorf("FromConfig(defaults) = %+v, want %+v", pal, want)
	}
}

// TestFromConfigResolution: [colors] values may name a map entry or carry a
// raw ANSI code (todo.cfg's literal \033 form); both resolve to the same
// bytes. The resolution lives in config; FromConfig snapshots the result.
func TestFromConfigResolution(t *testing.T) {
	body := "[colors]\n" +
		"pri_a = \"yellow\"\n" +
		"color_date = \"\\\\033[0;31m\"\n" +
		"color_project = \"red\"\n" +
		"color_meta = \"\"\n"
	pal := FromConfig(loadConfig(t, body))

	if pal.PriA != "\x1b[1;33m" {
		t.Errorf("PriA = %q, want %q (map name)", pal.PriA, "\x1b[1;33m")
	}
	if pal.Date != "\x1b[0;31m" {
		t.Errorf("Date = %q, want %q (raw ANSI)", pal.Date, "\x1b[0;31m")
	}
	if pal.Project != "\x1b[0;31m" {
		t.Errorf("Project = %q, want %q (map name)", pal.Project, "\x1b[0;31m")
	}
	if pal.Meta != "" {
		t.Errorf("Meta = %q, want empty (explicitly off)", pal.Meta)
	}
}

// TestColorRoles pins the role → field mapping and the empty result for
// roles outside the palette, matched case-insensitively like config.
func TestColorRoles(t *testing.T) {
	pal := Color{
		PriA: "a", PriB: "b", PriC: "c", PriX: "x",
		Done: "d", Project: "p", Context: "k", Date: "dt", Number: "n", Meta: "m",
	}
	for role, want := range map[string]string{
		"pri_a": "a", "pri_b": "b", "pri_c": "c", "pri_x": "x",
		"color_done": "d", "color_project": "p", "color_context": "k",
		"color_date": "dt", "color_number": "n", "color_meta": "m",
	} {
		if got := pal.Color(role); got != want {
			t.Errorf("Color(%s) = %q, want %q", role, got, want)
		}
	}
	for _, role := range []string{"PRI_A", "Color_Done"} {
		if got := pal.Color(role); got == "" {
			t.Errorf("Color(%s) = %q, want a case-insensitive match", role, got)
		}
	}
	for _, role := range []string{"", "color_bogus", "pri_d", "done"} {
		if got := pal.Color(role); got != "" {
			t.Errorf("Color(%s) = %q, want empty for an unknown role", role, got)
		}
	}
}

// TestPriorityColor pins todo.cfg's priority map: A/B/C come from their
// fields and every other letter falls back to PriX (t1330: pri D..Z render
// as pri_x).
func TestPriorityColor(t *testing.T) {
	pal := Color{PriA: "a", PriB: "b", PriC: "c", PriX: "x"}
	for _, letter := range []byte{'A', 'B', 'C'} {
		want := map[byte]string{'A': "a", 'B': "b", 'C': "c"}[letter]
		if got := pal.PriorityColor(letter); got != want {
			t.Errorf("PriorityColor(%c) = %q, want %q", letter, got, want)
		}
	}
	for _, letter := range []byte{'D', 'Z', 'a'} {
		if got := pal.PriorityColor(letter); got != "x" {
			t.Errorf("PriorityColor(%c) = %q, want pri_x %q", letter, got, "x")
		}
	}
}

// TestPlainMode: plain mode short-circuits in config, so the palette carries
// no escape sequences at all and renders nothing.
func TestPlainMode(t *testing.T) {
	t.Setenv("TODOTXT_PLAIN", "1")
	pal := FromConfig(loadConfig(t, "[colors]\npri_a = \"yellow\"\ncolor_project = \"red\"\n"))
	if !reflect.DeepEqual(pal, Color{}) {
		t.Errorf("plain palette = %+v, want all-empty %+v", pal, Color{})
	}
	for _, role := range []string{"pri_a", "color_done", "color_project"} {
		if got := pal.Color(role); got != "" {
			t.Errorf("Color(%s) = %q, want empty in plain mode", role, got)
		}
	}
	if got := pal.PriorityColor('A'); got != "" {
		t.Errorf("PriorityColor(A) = %q, want empty in plain mode", got)
	}
}

// TestZeroValuePlain: a zero Color is a plain palette — the same code path
// serves both modes.
func TestZeroValuePlain(t *testing.T) {
	var pal Color
	if got := pal.Color("color_done"); got != "" {
		t.Errorf("zero palette Color = %q, want empty", got)
	}
	if got := pal.PriorityColor('A'); got != "" {
		t.Errorf("zero palette PriorityColor = %q, want empty", got)
	}
}

// TestPipelineHighlighting runs the real listing pipeline with the palette
// (t1360/t1380 fixtures): each colored word is wrapped in its color with a
// DEFAULT + line-color reset after it, and a colored line closes with a
// final DEFAULT. Byte-identical to the real todo.sh output.
func TestPipelineHighlighting(t *testing.T) {
	t1360 := []string{
		"(A) prioritized @con01 context",
		"(B) prioritized +prj02 project",
		"(C) prioritized context at EOL @con03",
		"(D) prioritized project at EOL +prj04",
		"+prj05 non-prioritized project at BOL",
		"@con06 non-prioritized context at BOL",
		"multiple @con_ @texts and +pro_ +jects",
		"non-contexts: seti@home @ @* @(foo)",
		"non-projects: lost+found + +! +(bar)",
	}

	t.Run("colored", func(t *testing.T) {
		body := "[colors]\ncolor_context = \"\\\\033[1m\"\ncolor_project = \"\\\\033[2m\"\n"
		got := formatLines(t, t1360, FromConfig(loadConfig(t, body)))
		want := []string{
			"\x1b[1;33m1 (A) prioritized \x1b[1m@con01\x1b[0m\x1b[1;33m context\x1b[0m",
			"\x1b[0;32m2 (B) prioritized \x1b[2m+prj02\x1b[0m\x1b[0;32m project\x1b[0m",
			"\x1b[1;34m3 (C) prioritized context at EOL \x1b[1m@con03\x1b[0m\x1b[1;34m\x1b[0m",
			"\x1b[1;37m4 (D) prioritized project at EOL \x1b[2m+prj04\x1b[0m\x1b[1;37m\x1b[0m",
			"5 \x1b[2m+prj05\x1b[0m non-prioritized project at BOL",
			"6 \x1b[1m@con06\x1b[0m non-prioritized context at BOL",
			"7 multiple \x1b[1m@con_\x1b[0m \x1b[1m@texts\x1b[0m and \x1b[2m+pro_\x1b[0m \x1b[2m+jects\x1b[0m",
			"8 non-contexts: seti@home @ @* @(foo)",
			"9 non-projects: lost+found + +! +(bar)",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("pipeline = %#v\nwant      %#v", got, want)
		}
	})

	t.Run("plain", func(t *testing.T) {
		got := formatLines(t, t1360, Color{})
		want := []string{
			"1 (A) prioritized @con01 context",
			"2 (B) prioritized +prj02 project",
			"3 (C) prioritized context at EOL @con03",
			"4 (D) prioritized project at EOL +prj04",
			"5 +prj05 non-prioritized project at BOL",
			"6 @con06 non-prioritized context at BOL",
			"7 multiple @con_ @texts and +pro_ +jects",
			"8 non-contexts: seti@home @ @* @(foo)",
			"9 non-projects: lost+found + +! +(bar)",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("pipeline = %#v\nwant      %#v", got, want)
		}
	})

	t.Run("date-number-meta", func(t *testing.T) {
		body := "[colors]\ncolor_date = \"\\\\033[0;31m\"\n" +
			"color_meta = \"\\\\033[0;32m\"\ncolor_number = \"\\\\033[0;34m\"\n"
		got := formatLines(t, []string{
			"2018-11-11 task with date",
			"task with metadata due:2018-12-31",
			"task without date and without metadata",
		}, FromConfig(loadConfig(t, body)))
		want := []string{
			"\x1b[0;34m1\x1b[0m \x1b[0;31m2018-11-11\x1b[0m task with date",
			"\x1b[0;34m2\x1b[0m task with metadata \x1b[0;32mdue:2018-12-31\x1b[0m",
			"\x1b[0;34m3\x1b[0m task without date and without metadata",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("pipeline = %#v\nwant      %#v", got, want)
		}
	})
}
