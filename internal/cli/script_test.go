package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/guerrero/gtdo/internal/cli"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"gtdo": func() {
			os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
		},
	})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(e *testscript.Env) error {
			e.Setenv("PATH", os.Getenv("PATH"))
			e.Setenv("TZ", "UTC")
			e.Setenv("HOME", e.WorkDir)
			// Hermetic config search: points at a file that never exists, so
			// the search stops at defaults and never reaches /etc.
			e.Setenv("GTDO_CONFIG", filepath.Join(e.WorkDir, "gtdo-config.toml"))
			// $ESC expands to a real ESC byte in cmpenv'd want files, so color
			// tests can be written readably.
			e.Setenv("ESC", "\x1b")
			// GTDO_TEST_NOW pins gtdo's clock to the fake epoch of the shell
			// suite (TODO_TEST_TIME=1234500000 = 2009-02-13T04:40:00Z), so the
			// completion/report actions and opt-in UUID sessions produce exact
			// transcripts. Scripts override it to advance.
			e.Setenv("GTDO_TEST_NOW", "2009-02-13T04:40:00Z")
			return nil
		},
		RequireExplicitExec: true,
	})
}
