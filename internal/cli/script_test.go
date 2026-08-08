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
			return nil
		},
		RequireExplicitExec: true,
	})
}
