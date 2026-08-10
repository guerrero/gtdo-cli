package man_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestManPageIsCommitted guards against the committed page drifting from
// the command tree. It regenerates the page with SOURCE_DATE_EPOCH pinned
// to the date the committed page was generated under, so regeneration is
// deterministic and the comparison is byte for byte.
func TestManPageIsCommitted(t *testing.T) {
	committed, err := os.ReadFile("gtdo.1")
	if err != nil {
		t.Fatalf("man/gtdo.1 is not committed: %v (run: make man)", err)
	}

	cmd := exec.Command("go", "run", "./tools/genman")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "SOURCE_DATE_EPOCH="+committedEpoch(t, string(committed)))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go run ./tools/genman: %v\n%s", err, out)
	}

	regenerated, err := os.ReadFile("gtdo.1")
	if err != nil {
		t.Fatal(err)
	}

	if string(committed) != string(regenerated) {
		t.Error("man/gtdo.1 is stale; run: make man")
	}
}

// committedEpoch returns the SOURCE_DATE_EPOCH value that reproduces the
// date the committed page was generated under. The .TH line reads
// .TH GTDO 1 "2006-01-02" ..., and genman renders that date from the
// epoch it was given, so the epoch regenerates the same page byte for
// byte.
func committedEpoch(t *testing.T, page string) string {
	t.Helper()
	line, _, ok := strings.Cut(page, "\n")
	if !ok || !strings.HasPrefix(line, ".TH ") {
		t.Fatalf("man/gtdo.1 does not start with a .TH line")
	}
	_, rest, ok := strings.Cut(line, `"`)
	if !ok {
		t.Fatalf(".TH line carries no date: %q", line)
	}
	date, _, ok := strings.Cut(rest, `"`)
	if !ok || date == "" {
		t.Fatalf(".TH line carries no date: %q", line)
	}
	when, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("unparseable .TH date %q: %v", date, err)
	}
	return strconv.FormatInt(when.UTC().Unix(), 10)
}

// TestManPageHasTheSectionsCobraDoesNotEmit checks the hand-written
// sections of the wrapper template are all present.
func TestManPageHasTheSectionsCobraDoesNotEmit(t *testing.T) {
	data, err := os.ReadFile("gtdo.1")
	if err != nil {
		t.Fatal(err)
	}

	for _, section := range []string{
		".SH NAME", ".SH SYNOPSIS", ".SH DESCRIPTION", ".SH COMMANDS",
		".SH OPTIONS", ".SH CONFIGURATION", ".SH INTERACTIVE ADD", ".SH ENVIRONMENT", ".SH FILES", ".SH EXIT STATUS", ".SH SEE ALSO",
	} {
		if !strings.Contains(string(data), section) {
			t.Errorf("man/gtdo.1 is missing %s", section)
		}
	}

	page := string(data)
	start := strings.Index(page, ".SH INTERACTIVE ADD\n")
	if start < 0 {
		t.Fatal("man/gtdo.1 has no INTERACTIVE ADD section")
	}
	end := strings.Index(page[start+len(".SH INTERACTIVE ADD\n"):], ".SH OPTIONS\n")
	if end < 0 {
		t.Fatal("man/gtdo.1 has no bounded INTERACTIVE ADD section")
	}
	interactive := page[start : start+len(".SH INTERACTIVE ADD\n")+end]
	for _, text := range []string{".B \\-i\n", ".B \\-g\n", ".B --only\n"} {
		if !strings.Contains(interactive, text) {
			t.Errorf("INTERACTIVE ADD section is missing %q", text)
		}
	}
	for _, text := range []string{"configured\n.B todo_file\nand\n.B done_file", "gtdo add \\-i", "gtdo add \\-g"} {
		if !strings.Contains(interactive, text) {
			t.Errorf("man/gtdo.1 is missing %q", text)
		}
	}
}

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

// TestManPageDocumentsSigilAllowLists guards the configuration contract for
// restricting contexts and projects in task text mutations.
func TestManPageDocumentsSigilAllowLists(t *testing.T) {
	data, err := os.ReadFile("gtdo.1")
	if err != nil {
		t.Fatal(err)
	}

	page := string(data)
	for _, phrase := range []string{".SH CONFIGURATION", "behaviour", "allowedContexts", "allowedProjects", "@work", "+gtdo", "not allowed"} {
		if !strings.Contains(page, phrase) {
			t.Errorf("man/gtdo.1 is missing %q", phrase)
		}
	}
}

// TestManPageDocumentsEveryExitCode pins the exit-status contract: gtdo
// has exactly two exit codes, like todo.sh.
func TestManPageDocumentsEveryExitCode(t *testing.T) {
	data, err := os.ReadFile("gtdo.1")
	if err != nil {
		t.Fatal(err)
	}

	for _, code := range []string{"0", "1"} {
		if !strings.Contains(string(data), ".B "+code+"\n") {
			t.Errorf("man/gtdo.1 does not document exit code %s", code)
		}
	}
}
