package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestActionUsageLines pins the help-block seam Tasks 7-8 build on: a
// command's usage lines are its Use line plus one line per alias, and
// querying by an alias shows only that alias's line (todo.sh's actionUsage
// extracts just the matched block).
func TestActionUsageLines(t *testing.T) {
	cmd := &cobra.Command{
		Use:     `append NR "TEXT TO APPEND"`,
		Aliases: []string{"app"},
	}

	t.Run("by name shows every alias", func(t *testing.T) {
		got := actionUsageLines(cmd, "append")
		want := []string{`append NR "TEXT TO APPEND"`, `app NR "TEXT TO APPEND"`}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("line %d = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("by alias shows only the alias line", func(t *testing.T) {
		got := actionUsageLines(cmd, "app")
		want := []string{`app NR "TEXT TO APPEND"`}
		if len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

// TestActionBlock pins the shape of a help block: usage lines at four
// spaces, description at six, no trailing newline.
func TestActionBlock(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "shorthelp",
		Long: "List the one-line usage of all built-in actions.",
	}
	want := "    shorthelp\n      List the one-line usage of all built-in actions."
	if got := actionBlock(cmd, "shorthelp"); got != want {
		t.Errorf("actionBlock = %q, want %q", got, want)
	}
}
